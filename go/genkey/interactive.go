package genkey

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
	util "github.com/waterdragen/akl-ws/util"
	"github.com/wayneashleyberry/truecolor/pkg/color"
)

type GenkeyInteractive struct {
	conn     *websocket.Conn
	userData *UserData
	sp       *util.StringPrinter
}

func NewGenkeyInteractive(conn *websocket.Conn, userData *UserData) *GenkeyInteractive {
	return &GenkeyInteractive{
		conn:     conn,
		userData: userData,
		sp:       util.NewStringPrinter(),
	}
}

func (self *GenkeyInteractive) SendMessage(s string) {
	self.conn.WriteMessage(websocket.TextMessage, []byte(s))
}

func (self *GenkeyInteractive) FlushSp() {
	msg := self.sp.FlushAndTrim()
	self.SendMessage(msg)
}

func (self *GenkeyInteractive) CopyLayout(src Layout) Layout {
	var l Layout
	n := len(src.Keys)
	l.Keys = make([][]string, n)
	for i := range src.Keys {
		l.Keys[i] = make([]string, len(src.Keys[i]))
		copy(l.Keys[i], src.Keys[i])
	}
	l.Name = src.Name
	l.Total = src.Total

	l.Keymap = make(map[string]Pos)
	for k, v := range src.Keymap {
		l.Keymap[k] = v
	}
	l.Fingermap = make(map[Finger][]Pos)
	for k, v := range src.Fingermap {
		l.Fingermap[k] = v
	}
	l.Fingermatrix = make(map[Pos]Finger)
	for k, v := range src.Fingermatrix {
		l.Fingermatrix[k] = v
	}
	return l
}

func (self *GenkeyInteractive) printlayout(l *Layout, px, py int) {
	for y, row := range l.Keys {
		for x, k := range row {
			freq := float64(self.userData.Data.Letters[k]) / (l.Total * 1.2)
			pc := freq / 0.1 //percent
			log := math.Log(1+pc) * 255
			base := math.Round(0.3 * 255)
			c := color.Color(uint8(0.6*base+log), uint8(base+log), uint8(base+log))

			self.sp.MoveCursor(px+(2*x), py+y)
			self.sp.PrintColor(c, k)
		}
	}
}

func (self *GenkeyInteractive) printfreqpairpercent(l *Layout, f FreqPair) {
	self.sp.Print(fmt.Sprintf("%s %.1f%% ", f.Ngram, 100*float64(f.Count)/l.Total))
}

func (self *GenkeyInteractive) printsfbs(l *Layout) {
	interactive := &self.userData.Interactive
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)

	sfbs := genkeyLayout.ListSFBs(*l, false)
	rate := genkeyLayout.SFBs(*l, false)
	genkeyLayout.SortFreqList(sfbs)
	self.sp.MoveCursor(4+(interactive.LayoutWidth*2), 1)
	self.sp.Print(fmt.Sprintf("SFBs %.2f%%", 100*rate/l.Total))
	for i := 0; i <= 4; i++ {
		self.sp.MoveCursor(4+(interactive.LayoutWidth*2), 2+i)
		self.sp.Print(fmt.Sprintf(" %s %s", sfbs[2*i].Ngram, sfbs[(2*i)+1].Ngram))
	}
}

func (self *GenkeyInteractive) printworst(l *Layout) {
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)
	interactive := &self.userData.Interactive

	bgs := genkeyLayout.ListWorstBigrams(*l)
	genkeyLayout.SortFreqList(bgs)
	self.sp.MoveCursor(3+(interactive.LayoutWidth*2)+13, 1)
	self.sp.Print("Worst BGs")
	for i := 0; i <= 4; i++ {
		self.sp.MoveCursor(3+(interactive.LayoutWidth*2)+13, 2+i)
		self.sp.Print(fmt.Sprintf(" %s %s", bgs[2*i].Ngram, bgs[(2*i)+1].Ngram))
	}
}

func (self *GenkeyInteractive) printtrigrams(l *Layout) {
	tg := NewGenkeyLayout(self.conn, self.userData).FastTrigrams(l, 0)
	total := float64(tg.Alternates)
	total += float64(tg.Onehands)
	total += float64(tg.LeftInwardRolls)
	total += float64(tg.LeftOutwardRolls)
	total += float64(tg.RightInwardRolls)
	total += float64(tg.RightOutwardRolls)
	total += float64(tg.Redirects)
	self.sp.MoveCursor(1, 7)
	self.sp.Print("Trigrams")
	self.sp.MoveCursor(1, 8)
	x := 0
	y := 0
	for i, v := range []float64{float64(tg.LeftInwardRolls + tg.LeftOutwardRolls + tg.RightOutwardRolls + tg.RightInwardRolls), float64(tg.Alternates), float64(tg.Onehands), float64(tg.Redirects)} {
		var c color.Message
		if i == 0 {
			c = *c.Color(166, 188, 220)
		} else if i == 1 {
			c = *c.Color(162, 136, 227)
		} else if i == 2 {
			c = *c.Color(217, 90, 120)
		} else if i == 3 {
			c = *c.Color(45, 167, 130)
		}

		for pc := math.Ceil(100 * float64(v) / total); pc > 0; pc -= 1 {
			self.sp.PrintColor(&c, "=")

			x++
			if x > 19 {
				self.sp.MoveCursorDown(1)
				self.sp.MoveCursorBackward(x)
				x = 0
				y++
				if y > 4 {
					break
				}
			}
		}

	}
}

type psbl struct {
	pair      Pair
	score     float64
	potential float64
}

func (self *GenkeyInteractive) worsen(l Layout, is33 bool) {
	n := 1000
	i := 0
	var klen int
	if is33 {
		klen = 33
	} else {
		klen = 30
	}
	for i < n {
		x := rand.Intn(klen)
		y := rand.Intn(klen)
		if x == y {
			continue
		}
		var xrow int
		var xcol int
		var yrow int
		var ycol int
		if is33 {
			if x < 12 {
				xrow = 0
				xcol = x
			} else if x < 12+11 {
				xrow = 1
				xcol = x - 12
			} else {
				xrow = 2
				xcol = x - 12 - 11
			}
			if y < 12 {
				yrow = 0
				ycol = y
			} else if y < 12+11 {
				yrow = 1
				ycol = y - 12
			} else {
				yrow = 2
				ycol = y - 12 - 11
			}
		} else {
			if x < 10 {
				xrow = 0
				xcol = x
			} else if x < 20 {
				xrow = 1
				xcol = x - 10
			} else {
				xrow = 2
				xcol = x - 20
			}
		}
		px := self.userData.Interactive.Pins[xrow][xcol]
		py := self.userData.Interactive.Pins[yrow][ycol]
		if px == "#" || py == "#" {
			continue
		}
		kx := l.Keys[xrow][xcol]
		ky := l.Keys[yrow][ycol]
		if px == kx || px == ky || py == kx || py == ky {
			continue
		}
		p1 := l.Keymap[kx]
		p2 := l.Keymap[ky]
		NewGenkeyGenerate(self.conn, self.userData).Swap(&l, p1, p2)
		i = i + 1
	}
}

func (self *GenkeyInteractive) SuggestSwaps(l Layout, depth int, maxdepth int, p *psbl, wg *sync.WaitGroup) psbl {
	genkeyGenerate := NewGenkeyGenerate(self.conn, self.userData)
	s1 := genkeyGenerate.Score(l)

	var possibilities []psbl
	for r1 := 0; r1 < 3; r1++ {
		for r2 := 0; r2 < 3; r2++ {
			for c1 := 0; c1 < len(l.Keys[r1]); c1++ {
				for c2 := 0; c2 < len(l.Keys[r2]); c2++ {
					if c1 == c2 && r1 == r2 {
						continue
					}
					p1 := Pos{c1, r1}
					p2 := Pos{c2, r2}

					genkeyGenerate.Swap(&l, p1, p2)
					s2 := genkeyGenerate.Score(l)
					diff := s1 - s2
					if depth < maxdepth && diff > self.userData.Interactive.Threshold {
						if depth == 0 {
							possibilities = append(possibilities, psbl{Pair{p1, p2}, s2, s2})
							go self.SuggestSwaps(self.CopyLayout(l), depth+1, maxdepth, &possibilities[len(possibilities)-1], wg)
						} else {
							go self.SuggestSwaps(self.CopyLayout(l), depth+1, maxdepth, p, wg)
							if s2 < *&p.potential {
								*&p.potential = s2
							}
						}
						wg.Add(1)
					} else if depth == maxdepth {
						if s2 < *&p.potential {
							*&p.potential = s2
						}
					}
					genkeyGenerate.Swap(&l, p1, p2)
				}
			}
		}
	}
	if depth != 0 {
		wg.Done()
		return psbl{}
	} else {
		wg.Wait()
		if len(possibilities) == 0 {
			return psbl{}
		}
		top := s1
		topindex := 0
		for i, v := range possibilities {
			if v.potential < top {
				top = v.potential
				topindex = i
			}
		}
		return possibilities[topindex]
	}
}

func (self *GenkeyInteractive) message(s ...string) {
	var base int = self.sp.Height - 2
	var offset int
	for i, v := range s {
		offset = len(s) - (i + 1)
		self.sp.MoveCursor(0, base-offset)
		self.sp.Print(v)
	}
}

func (self *GenkeyInteractive) InteractiveSubsequent(input string) {
	defer self.FlushSp()

	self.sp.Clear()
	self.SendMessage("[CLEAR]")

	interactive := &self.userData.Interactive
	l := interactive.Layout
	args := strings.Fields(input)
	is33 := false
	noCross := true

	self.sp.MoveCursor(0, self.sp.Height-2)

	switch args[0] {
	case "t":
		var changeMessage string
		enabled := &self.userData.Config.Weights.Score.Trigrams.Enabled
		*enabled = !*enabled
		if *enabled {
			changeMessage = "enabled"
		} else {
			changeMessage = "disabled"
		}
		self.message(fmt.Sprintf("%s trigrams", changeMessage))
	case "s":
		if len(args) < 3 {
			self.message("usage: s key1 key2", "example: s a b")
			break
		}
		p1 := l.Keymap[args[1]]
		p2 := l.Keymap[args[2]]
		NewGenkeyGenerate(self.conn, self.userData).Swap(&l, p1, p2)
		interactive.Aswaps[0] = p1
		interactive.Bswaps[0] = p2
		interactive.Swapnum = 1
		self.message(fmt.Sprintf("swapped %s(%d,%d) with %s(%d,%d)", args[1], p1.Col, p1.Row, args[2], p2.Col, p2.Row))
	case "cs":
		if len(args) < 3 {
			self.message("usage: cs key1/co1 key2/col2", "examples: cs a b  ||  cs 0 1")
			break
		}
		var c1 int
		var c2 int
		if n, err := strconv.Atoi(args[1]); err == nil {
			c1 = n
		} else {
			c1 = l.Keymap[args[1]].Col
		}

		if n, err := strconv.Atoi(args[2]); err == nil {
			c2 = n
		} else {
			c2 = l.Keymap[args[2]].Col
		}
		for r := 0; r < 3; r++ {
			p1 := Pos{c1, r}
			p2 := Pos{c2, r}
			NewGenkeyGenerate(self.conn, self.userData).Swap(&l, p1, p2)
			interactive.Aswaps[r] = p1
			interactive.Bswaps[r] = p2
		}
		interactive.Swapnum = 3
		self.message(fmt.Sprintf("swapped c%d with c%d", c1, c2))
	case "r":
		for i := 0; i < interactive.Swapnum; i++ {
			NewGenkeyGenerate(self.conn, self.userData).Swap(&l, interactive.Aswaps[i], interactive.Bswaps[i])
		}
		self.message("reverted last swap")
	case "g":
		var max int
		if len(args) < 2 {
			max = 1
		} else {
			max, _ = strconv.Atoi(args[1])
			interactive.Threshold = 0
		}
		c := self.CopyLayout(l)
		var wg sync.WaitGroup
		swaps := self.SuggestSwaps(c, 0, max, &psbl{}, &wg)
		k1 := l.Keys[swaps.pair[0].Row][swaps.pair[0].Col]
		k2 := l.Keys[swaps.pair[1].Row][swaps.pair[1].Col]
		if swaps.score == 0.0 {
			self.message("no suggestion")
		} else {
			self.message(fmt.Sprintf("try %s (%.1f immediate, %.1f potential)", k1+k2, swaps.score, swaps.potential))
		}
	case "w":
		self.worsen(l, is33)
	case "m2":
		NewGenkeyLayout(self.conn, self.userData).MinimizeLayout(&l, interactive.Pins, 1, true, is33, noCross)
	case "m":
		NewGenkeyLayout(self.conn, self.userData).MinimizeLayout(&l, interactive.Pins, 0, true, is33, noCross)
	case "q":
		interactive.InInteractive = false
	case "save":
		self.message("Unsupported feature in demo mode")
	}

	self.printUpdatedLayout(time.Now())
	self.sp.Print(":")
}

func (self *GenkeyInteractive) InteractiveInitial(l Layout) {
	defer self.FlushSp()

	interactive := &self.userData.Interactive
	interactive.InInteractive = true
	interactive.Layout = l

	for _, row := range l.Keys {
		for x := range row {
			if x > interactive.LayoutWidth {
				interactive.LayoutWidth = x
			}
		}
	}
	self.sp.Clear()

	interactive.Aswaps = make([]Pos, 3)
	interactive.Bswaps = make([]Pos, 3)
	interactive.Swapnum = 0
	interactive.Pins = [][]string{
		{"@", "#", "#", "#", "@", "@", "#", "#", "#", "@", "#", "#"},
		{"#", "#", "#", "#", "@", "@", "#", "#", "#", "#", "#", "@"},
		{"@", "@", "@", "@", "@", "@", "@", "@", "@", "@", "@", "@"},
	}

	self.printUpdatedLayout(time.Now())
	self.sp.Print(":")
}

func (self *GenkeyInteractive) printUpdatedLayout(start time.Time) {
	l := self.userData.Interactive.Layout

	self.sp.MoveCursor(0, 0)
	self.sp.Print(l.Name)
	self.printlayout(&l, 1, 2)
	self.sp.MoveCursor(1, 5)
	self.sp.Print(fmt.Sprintf("Score: %.2f", NewGenkeyGenerate(self.conn, self.userData).Score(l)))
	self.printsfbs(&l)
	self.printworst(&l)
	self.printtrigrams(&l)
	end := time.Now()
	elapsed := end.Sub(start)
	millis := float64(elapsed) / float64(time.Millisecond)
	s := fmt.Sprintf("%vms", millis)
	self.sp.MoveCursor(self.sp.Width-len(s), 1)
	self.sp.Print(s)
	self.sp.MoveCursor(0, self.sp.Height-1)
}
