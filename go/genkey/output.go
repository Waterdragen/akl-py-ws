/*UserConfig
Copyright (C) 2024 semi
    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.
    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.
    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package genkey

import (
	//"strings"
	"fmt"
	"math"

	"github.com/fogleman/gg"
	websocket "github.com/gorilla/websocket"
	"github.com/wayneashleyberry/truecolor/pkg/color"
)

type GenkeyOutput struct {
	conn     *websocket.Conn
	userData *UserData
}

func NewGenkeyOutput(conn *websocket.Conn, userData *UserData) *GenkeyOutput {
	return &GenkeyOutput{conn, userData}
}

func (self *GenkeyOutput) SendMessage(s string) {
	self.conn.WriteMessage(websocket.TextMessage, []byte(s))
}

func (self *GenkeyOutput) PrintLayout(keys [][]string) {
	for _, row := range keys {
		for x, key := range row {
			self.SendMessage(fmt.Sprintf("%s ", key))
			if x == 4 {
				self.SendMessage(" ")
			}
		}
		self.SendMessage("\n")
	}
}

func (self *GenkeyOutput) PrintAnalysis(l Layout) {
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)

	color.White().Bold().Println(l.Name)
	self.PrintLayout(l.Keys)

	duplicates, missing := genkeyLayout.DuplicatesAndMissing(l)
	if len(duplicates) > 0 {
		println(len(duplicates))
		self.SendMessage(fmt.Sprintf("Duplicate characters: %s\n", duplicates))
	}
	if len(missing) > 0 {
		self.SendMessage(fmt.Sprintf("Missing characters: %s\n", missing))
	}

	ftri := genkeyLayout.FastTrigrams(&l, 0)
	ftotal := float64(ftri.Total)
	leftrolls := 100*float64(ftri.LeftInwardRolls)/ftotal + 100*float64(ftri.LeftOutwardRolls)/ftotal
	rightrolls := 100*float64(ftri.RightInwardRolls)/ftotal + 100*float64(ftri.RightOutwardRolls)/ftotal
	self.SendMessage(fmt.Sprintf("Rolls (l): %.2f%%\n", leftrolls))
	self.SendMessage(fmt.Sprintf("\tInward: %.2f%%\n", 100*float64(ftri.LeftInwardRolls)/ftotal))
	self.SendMessage(fmt.Sprintf("\tOutward: %.2f%%\n", 100*float64(ftri.LeftOutwardRolls)/ftotal))
	self.SendMessage(fmt.Sprintf("Rolls (r): %.2f%%\n", rightrolls))
	self.SendMessage(fmt.Sprintf("\tInward: %.2f%%\n", 100*float64(ftri.RightInwardRolls)/ftotal))
	self.SendMessage(fmt.Sprintf("\tOutward: %.2f%%\n", 100*float64(ftri.RightOutwardRolls)/ftotal))
	self.SendMessage(fmt.Sprintf("Alternates: %.2f%%\n", 100*float64(ftri.Alternates)/ftotal))
	self.SendMessage(fmt.Sprintf("Onehands: %.2f%%\n", 100*float64(ftri.Onehands)/ftotal))
	self.SendMessage(fmt.Sprintf("Redirects: %.2f%%\n", 100*float64(ftri.Redirects)/ftotal))

	var weighted []float64
	var unweighted []float64
	if self.userData.DynamicFlag {
		weighted = genkeyLayout.DynamicFingerSpeed(&l, true)
		unweighted = genkeyLayout.DynamicFingerSpeed(&l, false)
	} else {
		weighted = genkeyLayout.FingerSpeed(&l, true)
		unweighted = genkeyLayout.FingerSpeed(&l, false)
	}
	var highestUnweightedFinger string
	var highestUnweighted float64
	var utotal float64

	var highestWeightedFinger string
	var highestWeighted float64
	var wtotal float64
	for i := 0; i < 8; i++ {
		utotal += unweighted[i]
		if unweighted[i] > highestUnweighted {
			highestUnweighted = unweighted[i]
			highestUnweightedFinger = FingerNames[i]
		}

		wtotal += weighted[i]
		if weighted[i] > highestWeighted {
			highestWeighted = weighted[i]
			highestWeightedFinger = FingerNames[i]
		}
	}
	self.SendMessage(fmt.Sprintf("Finger Speed (weighted): %.2f\n", weighted))
	self.SendMessage(fmt.Sprintf("Finger Speed (unweighted): %.2f\n", unweighted))
	self.SendMessage(fmt.Sprintf("Highest Speed (weighted): %.2f (%s)\n", highestWeighted, highestWeightedFinger))
	self.SendMessage(fmt.Sprintf("Highest Speed (unweighted): %.2f (%s)\n", highestUnweighted, highestUnweightedFinger))
	left, right := genkeyLayout.IndexUsage(l)
	self.SendMessage(fmt.Sprintf("Index Usage: %.1f%% %.1f%%\n", left, right))
	var sfb float64
	var sfbs []FreqPair

	ngcount := self.userData.Config.Output.Analysis.TopNgrams
	if !self.userData.DynamicFlag {
		sfb = genkeyLayout.SFBs(l, false)
		sfbs = genkeyLayout.ListSFBs(l, false)
		self.SendMessage(fmt.Sprintf("SFBs: %.3f%%\n", 100*sfb/l.Total))
		self.SendMessage(fmt.Sprintf("DSFBs: %.3f%%\n", 100*genkeyLayout.SFBs(l, true)/l.Total))
		lsb := float64(genkeyLayout.LSBs(l))
		self.SendMessage(fmt.Sprintf("LSBs: %.2f%%\n", 100*lsb/l.Total))

		genkeyLayout.SortFreqList(sfbs)

		self.SendMessage("Top SFBs:\n")
		self.PrintFreqList(sfbs, ngcount, true)
	} else {
		sfb = genkeyLayout.DynamicSFBs(l)
		escaped, real := genkeyLayout.ListDynamic(l)
		self.SendMessage(fmt.Sprintf("Real SFBs: %.3f%%\n", 100*sfb/l.Total))
		self.PrintFreqList(real, 8, true)
		self.SendMessage("Dynamic Completions:\n")
		self.PrintFreqList(escaped, 30, true)
	}

	if !self.userData.DynamicFlag {
		bigrams := genkeyLayout.ListWorstBigrams(l)
		genkeyLayout.SortFreqList(bigrams)
		self.SendMessage("Worst Bigrams:\n")
		self.PrintFreqList(bigrams, ngcount, false)
	}

	self.SendMessage(fmt.Sprintf("Score: %.2f\n", NewGenkeyGenerate(self.conn, self.userData).Score(l)))
	self.SendMessage("\n")
}

func (self *GenkeyOutput) PrintFreqList(list []FreqPair, length int, percent bool) {
	pc := ""
	if percent {
		pc = "%"
	}
	for i, v := range list[0:length] {
		self.SendMessage(fmt.Sprintf("\t%s %.3f%s", v.Ngram, 100*float64(v.Count)/float64(self.userData.Data.TotalBigrams), pc))
		if (i+1)%4 == 0 {
			self.SendMessage("\n")
		}
	}
	self.SendMessage("\n")
}

func (self *GenkeyOutput) Heatmap(layout Layout) {
	l := layout.Keys
	dc := gg.NewContext(500, 160)

	cols := []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	for row, r := range l {
		for col, c := range r {
			if col > 9 {
				continue
			}
			dc.DrawRectangle(float64(50*col), float64(50*row), 50, 50)
			freq := float64(self.userData.Data.Letters[c]) / (layout.Total * 1.15)
			cols[col] += freq
			pc := freq / 0.1 //percent
			log := math.Log(1 + pc)
			base := 0.3
			dc.SetRGB(0.6*(base+log), base*(1-pc), base+log)
			dc.Fill()
			dc.SetRGB(0, 0, 0)
			dc.DrawString(c, 22.5+float64(50*col), 27.5+float64(50*row))
		}
	}

	for i, c := range cols {
		dc.DrawRectangle(float64(50*i), 150, 50, 10)
		pc := c / 0.2
		log := math.Log(1 + pc)
		base := 0.3
		dc.SetRGB(0.6*(base+log), base*(1-pc), base+log)
		dc.Fill()
	}

	dc.SavePNG(self.userData.Config.Paths.Heatmap)
}
