/*
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
	"fmt"
	"math"
	"math/rand"
	"sort"

	"strings"
	"time"

	websocket "github.com/gorilla/websocket"
)

type GenkeyGenerate struct {
	conn     *websocket.Conn
	userData *UserData
}

func NewGenkeyGenerate(conn *websocket.Conn, userData *UserData) *GenkeyGenerate {
	return &GenkeyGenerate{conn, userData}
}

func (self *GenkeyGenerate) SendMessage(s string) {
	self.conn.WriteMessage(websocket.TextMessage, []byte(s))
}

// Max Rolls: 30%

func (self *GenkeyGenerate) Score(l Layout) float64 {
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)

	var score float64
	s := &self.userData.Config.Weights.Score
	if s.FSpeed != 0 {
		var speeds []float64
		if !self.userData.DynamicFlag {
			speeds = genkeyLayout.FingerSpeed(&l, true)
		} else {
			speeds = genkeyLayout.DynamicFingerSpeed(&l, true)
		}
		total := 0.0
		for _, s := range speeds {
			total += s
		}
		score += s.FSpeed * total
	}
	if s.LSB != 0 {
		score += s.LSB * 100 * float64(genkeyLayout.LSBs(l)) / l.Total
	}
	if s.Trigrams.Enabled {
		tri := genkeyLayout.FastTrigrams(&l, s.Trigrams.Precision)
		score += s.Trigrams.LeftInwardRoll * (100 - (100 * float64(tri.LeftInwardRolls) / float64(tri.Total)))
		score += s.Trigrams.RightInwardRoll * (100 - (100 * float64(tri.RightInwardRolls) / float64(tri.Total)))
		score += s.Trigrams.LeftOutwardRoll * (100 - (100 * float64(tri.LeftOutwardRolls) / float64(tri.Total)))
		score += s.Trigrams.RightOutwardRoll * (100 - (100 * float64(tri.RightOutwardRolls) / float64(tri.Total)))
		score += s.Trigrams.Alternate * (100 - (100 * float64(tri.Alternates) / float64(tri.Total)))
		score += s.Trigrams.Onehand * (100 - (100 * float64(tri.Onehands) / float64(tri.Total)))
		score += s.Trigrams.Redirect * (100 * float64(tri.Redirects) / float64(tri.Total))
	}

	if s.IndexBalance != 0 {
		left, right := genkeyLayout.IndexUsage(l)
		score += s.IndexBalance * math.Abs(right-left)
	}

	self.userData.Analyzed++

	return score
}

func (self *GenkeyGenerate) randomLayout() Layout {
	chars := self.userData.Config.Generation.GeneratedLayoutChars
	var k [][]string
	k = make([][]string, 3)
	var l Layout
	for row := 0; row < 3; row++ {
		k[row] = make([]string, 10)
		for col := 0; col < 10; col++ {
			char := string([]rune(chars)[rand.Intn(len(chars))])
			k[row][col] += char
			l.Total += float64(self.userData.Data.Letters[char])
			chars = strings.Replace(chars, char, "", 1)
		}
	}

	l.Keys = k
	l.Keymap = NewGenkeyLayout(self.conn, self.userData).GenKeymap(k)
	l.Fingermap = self.userData.GeneratedFingermap
	l.Fingermatrix = self.userData.GeneratedFingermatrix

	return l
}

type layoutScore struct {
	l     Layout
	score float64
}

func (self *GenkeyGenerate) sortLayouts(layouts []layoutScore) {
	sort.Slice(layouts, func(i, j int) bool {
		var iscore float64
		var jscore float64
		if layouts[i].score != 0 {
			iscore = layouts[i].score
		} else {
			iscore = self.Score(layouts[i].l)
			layouts[i].score = iscore
		}

		if layouts[j].score != 0 {
			jscore = layouts[j].score
		} else {
			jscore = self.Score(layouts[j].l)
			layouts[j].score = jscore
		}
		return iscore < jscore
	})
}

func (self *GenkeyGenerate) Populate(n int) Layout {
	layouts := []layoutScore{}
	for i := 0; i < n; i++ {
		if !self.userData.ImproveFlag {
			layouts = append(layouts, layoutScore{self.randomLayout(), 0})
		} else {
			layouts = append(layouts, layoutScore{NewGenkeyInteractive(self.conn, self.userData).CopyLayout(self.userData.ImproveLayout), 0})
		}

	}
	self.SendMessage(fmt.Sprintf("%d random created...\r\n", n))

	for i := range layouts {
		layouts[i].score = 0
		go self.greedyImprove(&layouts[i].l)
	}

	analyzed := 0
	goroCounter := &self.userData.GoroutineCounter

	for goroCounter.GetCount() > 1 {
		self.SendMessage(fmt.Sprintf("%d greedy improving at %d analyzed/s       \r", goroCounter.GetCount()-1, self.userData.Analyzed-analyzed))
		analyzed = self.userData.Analyzed
		time.Sleep(time.Second)
	}

	goroCounter.Reset()

	self.SendMessage("\n")

	self.SendMessage("Sorting...\n")
	self.sortLayouts(layouts)

	genkeyOutput := NewGenkeyOutput(self.conn, self.userData)
	genkeyOutput.PrintLayout(layouts[0].l.Keys)
	self.SendMessage(fmt.Sprintf("%v\n", self.Score(layouts[0].l)))
	genkeyOutput.PrintLayout(layouts[1].l.Keys)
	self.SendMessage(fmt.Sprintf("%v\n", self.Score(layouts[1].l)))
	genkeyOutput.PrintLayout(layouts[2].l.Keys)
	self.SendMessage(fmt.Sprintf("%v\n", self.Score(layouts[2].l)))

	layouts = layouts[0:self.userData.Config.Generation.Selection]

	for i := range layouts {
		layouts[i].score = 0
		go self.fullImprove(&layouts[i].l)
	}

	for goroCounter.GetCount() > 1 {
		self.SendMessage(fmt.Sprintf("%d fully improving at %d analyzed/s      \r", goroCounter.GetCount()-1, self.userData.Analyzed-analyzed))
		analyzed = self.userData.Analyzed
		time.Sleep(time.Second)
	}
	goroCounter.Reset()

	self.sortLayouts(layouts)

	self.SendMessage("\n")
	best := layouts[0]

	for col := 0; col < 10; col++ {
		if col >= 3 && col <= 6 {
			continue
		}
		if self.userData.Data.Letters[best.l.Keys[0][col]] < self.userData.Data.Letters[best.l.Keys[2][col]] {
			self.Swap(&best.l, Pos{col, 0}, Pos{col, 2})
		}
	}

	genkeyOutput.PrintAnalysis(best.l)
	if self.userData.Config.Output.Generation.Heatmap {
		genkeyOutput.Heatmap(best.l)
	}

	//improved := ImproveRedirects(layouts[0].keys)
	//PrintAnalysis("Generated (improved redirects)", improved)
	//Heatmap(improved)

	return layouts[0].l
}

func (self *GenkeyGenerate) RandPos() Pos {
	var p Pos
	if self.userData.ImproveFlag {
		n := len(self.userData.SwapPossibilities)
		p = self.userData.SwapPossibilities[rand.Intn(n)]
	} else {
		col := rand.Intn(10)
		row := rand.Intn(3)
		p = Pos{col, row}
	}
	return p
}

func (self *GenkeyGenerate) greedyImprove(layout *Layout) {
	self.userData.GoroutineCounter.Increment()
	defer self.userData.GoroutineCounter.Decrement()

	stuck := 0
	for {
		first := self.Score(*layout)

		a := self.RandPos()
		b := self.RandPos()
		self.Swap(layout, a, b)

		second := self.Score(*layout)

		if second < first {
			// accept
			stuck = 0
		} else {
			self.Swap(layout, a, b)
			stuck++
		}

		if stuck > 500 {
			return
		}

	}
}

func (self *GenkeyGenerate) fullImprove(layout *Layout) {
	self.userData.GoroutineCounter.Increment()
	defer self.userData.GoroutineCounter.Decrement()

	i := 0
	tier := 2
	changed := false
	changes := 0
	rejected := 0
	max := 600
	Swaps := make([]Pair, 7)
	for {
		i += 1
		first := self.Score(*layout)

		for j := tier - 1; j >= 0; j-- {
			a := self.RandPos()
			b := self.RandPos()
			self.Swap(layout, a, b)
			Swaps[j] = Pair{a, b}
		}

		second := self.Score(*layout)

		if second < first {
			i = 0
			changed = true
			changes++
			continue
		} else {
			for j := 0; j < tier; j++ {
				self.Swap(layout, Swaps[j][0], Swaps[j][1])
			}

			rejected++

			if i > max {
				if changed {
					tier = 1
				} else {
					tier++
				}

				max = 900 * tier * tier

				changed = false

				if tier > 3 {
					break
				}

				i = 0
			}
		}
		continue
	}

}

func (self *GenkeyGenerate) Swap(l *Layout, a, b Pos) {
	k := l.Keys
	m := l.Keymap
	k[a.Row][a.Col], k[b.Row][b.Col] = k[b.Row][b.Col], k[a.Row][a.Col]
	m[k[a.Row][a.Col]] = a
	m[k[b.Row][b.Col]] = b

	l.Keys = k
	l.Keymap = m
}
