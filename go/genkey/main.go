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
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	websocket "github.com/gorilla/websocket"
	"github.com/wayneashleyberry/truecolor/pkg/color"
)

type Argument int

const (
	NullArg Argument = iota
	LayoutArg
	NgramArg
	PathArg
)

type Command struct {
	Names       []string
	Description string
	Arg         Argument
	CountArg    bool
}

var Commands = []Command{
	{
		Names:       []string{"load"},
		Description: "loads a text file as a corpus",
		Arg:         PathArg,
	},
	{
		Names:       []string{"rank", "r"},
		Description: "returns a ranked list of layouts",
		Arg:         NullArg,
	},
	{
		Names:       []string{"analyze", "a"},
		Description: "outputs detailed analysis of a layout",
		Arg:         LayoutArg,
	},
	{
		Names:       []string{"interactive"},
		Description: "enters interactive analysis mode for the given layout",
		Arg:         LayoutArg,
	},
	{
		Names:       []string{"generate", "g"},
		Description: "attempts to generate an optimal layout based on weights.hjson",
		Arg:         NullArg,
	},
	{
		Names:       []string{"improve"},
		Description: "attempts to improve a layout according to the restrictions in layouts/_generate",
		Arg:         LayoutArg,
	},
	{
		Names:       []string{"heatmap"},
		Description: "outputs a heatmap for the given layout at heatmap.png",
		Arg:         LayoutArg,
	},
	{
		Names:       []string{"sfbs"},
		Description: "lists the sfb frequency and most frequent sfbs",
		Arg:         LayoutArg,
		CountArg:    true,
	},
	{
		Names:       []string{"dsfbs"},
		Description: "lists the dsfb frequency and most frequent dsfbs",
		Arg:         LayoutArg,
		CountArg:    true,
	},
	{
		Names:       []string{"lsbs"},
		Description: "lists the lsb frequency and most frequent lsbs",
		Arg:         LayoutArg,
		CountArg:    true,
	},
	{
		Names:       []string{"speed"},
		Description: "lists each finger and its unweighted speed",
		Arg:         LayoutArg,
		CountArg:    true,
	},
	{
		Names:       []string{"bigrams"},
		Description: "lists the worst key pair relationships",
		Arg:         LayoutArg,
		CountArg:    true,
	},
	{
		Names:       []string{"ngram"},
		Description: "lists the frequency of a given ngram",
		Arg:         NgramArg,
	},
}

type GenkeyMain struct {
	conn     *websocket.Conn
	userData *UserData
}

func NewGenkeyMain(conn *websocket.Conn, cachedUserData *UserData) *GenkeyMain {
	var userData *UserData
	if cachedUserData == nil {
		userData = &UserData{}
	} else {
		userData = cachedUserData
	}

	return &GenkeyMain{
		conn:     conn,
		userData: userData,
	}
}

func (self *GenkeyMain) SendMessage(s string) {
	self.conn.WriteMessage(websocket.TextMessage, []byte(s))
}

func (self *GenkeyMain) getLayout(s string) *Layout {
	s = strings.ToLower(s)
	if l, ok := self.userData.Layouts[s]; ok {
		return l
	}
	self.SendMessage(fmt.Sprintf("layout [%s] was not found\n", s))
	return nil
}

func (self *GenkeyMain) runCommand(args []string) {
	var layout *Layout
	var ngram *string
	var cmd string
	count := 0

	if len(args) == 0 {
		self.usage()
		return
	}

	for _, command := range Commands {
		matches := false
		for _, name := range command.Names {
			if name == args[0] {
				matches = true
				break
			}
		}
		if !matches {
			continue
		}
		cmd = command.Names[0]
		if command.Arg == NullArg {
			break
		}
		if len(args) == 1 {
			self.commandUsage(&command)
			return
		}
		if command.Arg == PathArg {
			if _, err := os.Stat(args[1]); errors.Is(err, os.ErrNotExist) {
				self.SendMessage(fmt.Sprintf("file [%s] does not exist\n", args[1]))
				return
			}
		} else if command.Arg == NgramArg {
			ngram = &args[1]
		} else if command.Arg == LayoutArg {
			layout = self.getLayout(args[1])
			if layout == nil {
				return
			}
		}
		if command.CountArg && len(args) == 3 {
			num, err := strconv.Atoi(args[2])
			if err != nil {
				self.SendMessage(fmt.Sprintf("optional count argument must be a number, not [%s]\n", args[2]))
				return
			}
			count = num
		}
		break
	}
	if cmd == "" {
		self.usage()
	}
	if cmd == "load" {
		self.SendMessage("Unsupported command in demo mode")

		//genkeyText := GenkeyText{self.conn}
		//Data = genkeyText.GetTextData(*path)
		//name := filepath.Base(*path)
		//name = name[:len(name)-len(filepath.Ext(name))]
		//name = name + ".json"
		//outpath := filepath.Join(self.userData.Config.Paths.Corpora, name)
		//println(outpath)
		//genkeyText.WriteData(Data, outpath)
	} else if cmd == "rank" {
		type x struct {
			name  string
			score float64
		}

		var sorted []x

		for _, v := range self.userData.Layouts {
			sorted = append(sorted, x{v.Name, NewGenkeyGenerate(self.conn, self.userData).Score(v)})
		}

		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].score < sorted[j].score
		})

		for _, l := range sorted {
			spaces := strings.Repeat(self.userData.Config.Output.Rank.Spacer, 1+self.userData.LongestLayoutName-len(l.name))
			self.SendMessage(fmt.Sprintf("%s%s%.2f\n", l.name, spaces, l.score))
		}
	} else if cmd == "analyze" {
		NewGenkeyOutput(self.conn, self.userData).PrintAnalysis(layout)
	} else if cmd == "generate" {
		genkeyGenerate := NewGenkeyGenerate(self.conn, self.userData)
		best := genkeyGenerate.Populate(self.userData.Config.Generation.InitialPopulation)
		optimal := genkeyGenerate.Score(best)

		type x struct {
			name  string
			score float64
		}

		var sorted []x

		for k, v := range self.userData.Layouts {
			sorted = append(sorted, x{k, genkeyGenerate.Score(v)})
		}

		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].score < sorted[j].score
		})

		for _, l := range sorted {
			spaces := strings.Repeat(self.userData.Config.Output.Rank.Spacer, 1+self.userData.LongestLayoutName-len(l.name))
			self.SendMessage(
				fmt.Sprintf("%s%s%d%%\n", l.name, spaces, int(100*optimal/(genkeyGenerate.Score(self.userData.Layouts[l.name])))),
			)
		}
	} else if cmd == "interactive" {
		NewGenkeyInteractive(self.conn, self.userData).InteractiveInitial(layout)

	} else if cmd == "heatmap" {
		self.SendMessage("Unsupported command in demo mode")
		// NewGenkeyOutput(self.conn, self.userData).Heatmap(*layout)
	} else if cmd == "improve" {
		genkeyGenerate := NewGenkeyGenerate(self.conn, self.userData)
		self.userData.ImproveFlag = true
		self.userData.ImproveLayout = layout
		best := genkeyGenerate.Populate(1000)
		optimal := genkeyGenerate.Score(best)

		self.SendMessage(
			fmt.Sprintf("%s %d%%\n", layout.Name, int(100*optimal/(NewGenkeyGenerate(self.conn, self.userData).Score(self.userData.ImproveLayout)))),
		)
	} else if cmd == "sfbs" || cmd == "dsfbs" || cmd == "lsbs" || cmd == "bigrams" {
		genkeyLayout := NewGenkeyLayout(self.conn, self.userData)
		var total float64
		var list []FreqPair
		if cmd == "sfbs" {
			total = 100 * float64(genkeyLayout.SFBs(layout, false)) / layout.Total
			list = genkeyLayout.ListSFBs(layout, false)
		} else if cmd == "dsfbs" {
			total = 100 * float64(genkeyLayout.SFBs(layout, true)) / layout.Total
			list = genkeyLayout.ListSFBs(layout, true)

		} else if cmd == "lsbs" {
			total = 100 * float64(genkeyLayout.LSBs(layout)) / layout.Total
			list = genkeyLayout.ListLSBs(layout)
		} else if cmd == "bigrams" {
			total = 0.0
			list = genkeyLayout.ListWorstBigrams(layout)
		}
		genkeyLayout.SortFreqList(list)
		if count == 0 {
			count = self.userData.Config.Output.Misc.TopNgrams
		}
		if total != 0.0 {
			self.SendMessage(fmt.Sprintf("%.2f%%\n", total))
		}
		NewGenkeyOutput(self.conn, self.userData).PrintFreqList(list, count, true)
	} else if cmd == "speed" {
		genkeyLayout := NewGenkeyLayout(self.conn, self.userData)
		unweighted := genkeyLayout.FingerSpeed(layout, false)
		self.SendMessage("Unweighted Speed\n")
		for i, v := range unweighted {
			self.SendMessage(fmt.Sprintf("\t%s: %.2f\n", FingerNames[i], v))
		}

		weighted := genkeyLayout.FingerSpeed(layout, true)
		self.SendMessage("Weighted Speed\n")
		for i, v := range weighted {
			self.SendMessage(fmt.Sprintf("\t%s: %.2f\n", FingerNames[i], v))
		}
	} else if cmd == "ngram" {
		total := float64(self.userData.Data.Total)
		ngram := *ngram
		switch len(ngram) {
		case 1:
			self.SendMessage(fmt.Sprintf("unigram: %.3f%%\n", 100*float64(self.userData.Data.Letters[ngram])/total))
		case 2:
			self.SendMessage(fmt.Sprintf("bigram: %.3f%%\n", 100*float64(self.userData.Data.Bigrams[ngram])/total))
			self.SendMessage(fmt.Sprintf("skipgram: %.3f%%\n", 100*self.userData.Data.Skipgrams[ngram]/total))
		case 3:
			self.SendMessage(fmt.Sprintf("trigram: %.3f%%\n", 100*float64(self.userData.Data.Trigrams[ngram])/total))
		default:
			self.SendMessage("Unimplemented feature in original app")
		}
	}
}

func (self *GenkeyMain) commandUsage(command *Command) {
	var argstr string
	if command.Arg == LayoutArg {
		argstr = " layout"
	} else if command.Arg == NgramArg {
		argstr = " ngram"
	} else if command.Arg == PathArg {
		argstr = " filepath"
	}

	argstr = color.White().Italic().Sprint(argstr)

	var countstr string
	if command.CountArg {
		countstr = " (count)"
	}

	self.SendMessage(fmt.Sprintf("%s%s%s | %s\n", command.Names[0], argstr, countstr, command.Description))
}

func (self *GenkeyMain) Run(input string) {
	self.userData.mu.Lock()
	defer self.userData.mu.Unlock()

	fs := flag.NewFlagSet("myProgram", flag.ExitOnError)
	args := strings.Fields(input)
	userData := self.userData

	ReadWeights(&userData.Config)
	fs.BoolVar(&userData.StaggerFlag, "stagger", userData.Config.Weights.Stagger, "if true, calculates distance for ANSI row-stagger form factor")
	fs.BoolVar(&userData.ColStaggerFlag, "colstagger", userData.Config.Weights.ColStagger, "if true, calculates distance for col-stagger form factor")
	fs.BoolVar(&userData.SlideFlag, "slide", false, "if true, ignores slideable sfbs (made for Oats) (might not work)")
	fs.BoolVar(&userData.DynamicFlag, "dynamic", false, "")
	fs.Parse(args)
	args = fs.Args()

	self.userData.Data = NewGenkeyText(self.conn, self.userData).LoadData(filepath.Join(self.userData.Config.Paths.Corpora, self.userData.Config.Corpus) + ".json")

	self.userData.Layouts = make(map[string]*Layout)
	NewGenkeyLayout(self.conn, self.userData).LoadLayoutDir()

	for _, l := range self.userData.Layouts {
		if len(l.Name) > self.userData.LongestLayoutName {
			self.userData.LongestLayoutName = len(l.Name)
		}
	}

	self.runCommand(args)
}

func (self *GenkeyMain) usage() {
	self.SendMessage("usage: genkey command argument (optional)\n")
	self.SendMessage("commands:\n")
	for _, c := range Commands {
		self.SendMessage("  ")
		self.commandUsage(&c)
	}
}

func (self *GenkeyMain) GetUserData() *UserData {
	return self.userData
}
