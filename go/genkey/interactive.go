package genkey

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/eiannone/keyboard"

	websocket "github.com/gorilla/websocket"
	util "github.com/waterdragen/akl-ws/util"
	"github.com/wayneashleyberry/truecolor/pkg/color"
)

type GenkeyInteractive struct {
	conn        *websocket.Conn
	userData    *UserData
	layoutwidth int
	sp          *util.StringPrinter
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
	msg := self.sp.Flush()
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
			self.sp.Print(c.Sprint(k))
		}
	}
}

func (self *GenkeyInteractive) printfreqpairpercent(l *Layout, f FreqPair) {
	self.sp.Print(fmt.Sprintf("%s %.1f%% ", f.Ngram, 100*float64(f.Count)/l.Total))
}

func (self *GenkeyInteractive) printsfbs(l *Layout) {
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)

	sfbs := genkeyLayout.ListSFBs(*l, false)
	rate := genkeyLayout.SFBs(*l, false)
	genkeyLayout.SortFreqList(sfbs)
	self.sp.MoveCursor(4+(self.layoutwidth*2), 1)
	self.sp.Print(fmt.Sprintf("SFBs %.2f%%", 100*rate/l.Total))
	for i := 0; i <= 4; i++ {
		self.sp.MoveCursor(4+(self.layoutwidth*2), 2+i)
		self.sp.Print(fmt.Sprintf(" %s %s", sfbs[2*i].Ngram, sfbs[(2*i)+1].Ngram))
	}
}

func (self *GenkeyInteractive) printworst(l *Layout) {
	genkeyLayout := NewGenkeyLayout(self.conn, self.userData)

	bgs := genkeyLayout.ListWorstBigrams(*l)
	genkeyLayout.SortFreqList(bgs)
	self.sp.MoveCursor(3+(self.layoutwidth*2)+13, 1)
	self.sp.Print("Worst BGs")
	for i := 0; i <= 4; i++ {
		self.sp.MoveCursor(3+(self.layoutwidth*2)+13, 2+i)
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
			s := c.Sprint("=")
			self.sp.Print(s)

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

type lScore struct {
	l Layout
	s float64
}

func (self *GenkeyInteractive) anneal(l Layout) {
	self.message("annealing...")
	self.FlushSp()

	rand.Seed(time.Now().Unix())

	genkeyGenerate := NewGenkeyGenerate(self.conn, self.userData)
	currentscore := genkeyGenerate.Score(l)

	x := int(float64(self.sp.Width)/2) - self.layoutwidth
	y := int(float64(self.sp.Height) / 2)

	self.printlayout(&l, x, y)
	self.FlushSp()

	for temp := 100; temp > 0; temp-- {
		self.message(fmt.Sprintf("annealing... %d degrees", temp))
		self.FlushSp()
		for i := 0; i < 2*(100-temp); i++ {
			p1 := genkeyGenerate.RandPos()
			p2 := genkeyGenerate.RandPos()
			genkeyGenerate.Swap(&l, p1, p2)
			s := genkeyGenerate.Score(l)
			if s < currentscore || rand.Intn(100) < temp {
				// accept
				currentscore = s

				self.printlayout(&l, x, y)
				self.FlushSp()
			} else {
				// reject
				genkeyGenerate.Swap(&l, p1, p2)
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
		px := pins[xrow][xcol]
		py := pins[yrow][ycol]
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

var threshold float64

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
					if depth < maxdepth && diff > threshold {
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
	self.sp.MoveCursor(0, self.sp.Height-2)
	blank := strings.Repeat("     ", 9)
	self.sp.Print(blank)
	for i, v := range s {
		self.sp.MoveCursor(0, self.sp.Height-(len(s)-i))
		self.sp.Print(v + blank)
	}
	self.FlushSp()
}

func (self *GenkeyInteractive) input() string {
	// TODO: rewrite a for loop into a single return from client after opening interactive mode

	var runes []rune
	self.sp.Print(fmt.Sprintf("%s\r", strings.Repeat(" ", self.sp.Width-2)))
	self.sp.Print(":")
	for {
		self.FlushSp()
		char, key, _ := keyboard.GetSingleKey()
		if key == keyboard.KeyEnter {
			break
		} else if key == keyboard.KeyBackspace || key == keyboard.KeyBackspace2 {
			if len(runes) > 0 {
				runes = runes[:len(runes)-1]

				self.sp.MoveCursorBackward(1)
				self.sp.Print("  ")
			}
		} else {
			if len(runes) >= self.sp.Width-1 {
				continue
			}
			if key == keyboard.KeySpace {
				char = ' '
			}
			runes = append(runes, char)
		}
		self.sp.MoveCursor(2, self.sp.Height)
		self.sp.Print(string(runes))
	}
	input := strings.TrimSpace(string(runes))
	return input
}

var pins [][]string

func (self *GenkeyInteractive) Interactive(l Layout) {

	/*
		TODO
		Cached data:
			- instance of Config
			- `awaps      []Pos`
			- `bswaps     []Pos`
			- `swapnum    int`
			- `pins       [][]string`
			- `threshold` float64
	*/
	for _, row := range l.Keys {
		for x := range row {
			if x > self.layoutwidth {
				self.layoutwidth = x
			}
		}
	}
	self.sp.Clear()
	self.SendMessage("[CLEAR]")

	aswaps := make([]Pos, 3)
	bswaps := make([]Pos, 3)
	var swapnum int

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		_ = keyboard.Close()
	}()

	pins = [][]string{
		{"@", "#", "#", "#", "@", "@", "#", "#", "#", "@", "#", "#"},
		{"#", "#", "#", "#", "@", "@", "#", "#", "#", "#", "#", "@"},
		{"@", "@", "@", "@", "@", "@", "@", "@", "@", "@", "@", "@"},
	}

	start := time.Now()

	// TODO: rewrite for loop as single execution
	// TODO: keep track of (any) interactive mode caches
	for {
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
		s := elapsed.String()
		self.sp.MoveCursor(self.sp.Width-len(s)-1, 1)
		self.sp.Print("  " + s)
		self.sp.MoveCursor(0, self.sp.Height)

		self.FlushSp()

		i := self.input()
		args := strings.Split(i, " ")

		start = time.Now()
		is33 := false
		noCross := true

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
			aswaps[0] = p1
			bswaps[0] = p2
			swapnum = 1
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
				aswaps[r] = p1
				bswaps[r] = p2
			}
			swapnum = 3
			self.message(fmt.Sprintf("swapped c%d with c%d", c1, c2))
		case "r":
			for i := 0; i < swapnum; i++ {
				NewGenkeyGenerate(self.conn, self.userData).Swap(&l, aswaps[i], bswaps[i])
			}
			self.message("reverted last swap")
		case "g":
			var max int
			if len(args) < 2 {
				max = 1
			} else {
				max, _ = strconv.Atoi(args[1])
				threshold = 0
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
			NewGenkeyLayout(self.conn, self.userData).MinimizeLayout(&l, pins, 1, true, is33, noCross)
		case "m":
			NewGenkeyLayout(self.conn, self.userData).MinimizeLayout(&l, pins, 0, true, is33, noCross)
		case "q":
			os.Exit(0)
		case "save":
			// TODO: disable this command

			self.message("enter a layout name:")
			self.FlushSp()
			name := self.input()
			filename := strings.ReplaceAll(name, " ", "_")
			filename = strings.ToLower(filename)
			filepath := path.Join("layouts", filename)
			_, err := os.Stat(filepath)
			if !os.IsNotExist(err) {
				self.message("this layout name is taken.", "are you sure you want to overwrite? (y/n)")
				self.FlushSp()
				i := self.input()
				self.message("", "")

				if i != "y" {
					break
				}
			}
			content := make([]string, 8)
			content[0] = name
			content[1] = strings.Join(l.Keys[0], " ")
			content[2] = strings.Join(l.Keys[1], " ")
			content[3] = strings.Join(l.Keys[2], " ")

			fingermatrix := make([][]string, 3)
			for i := 0; i < 3; i++ {
				fingermatrix[i] = make([]string, 20)
			}

			for p, n := range l.Fingermatrix {
				fingermatrix[p.Row][p.Col] = strconv.Itoa(int(n))
			}
			content[4] = strings.Join(fingermatrix[0], " ")
			content[5] = strings.Join(fingermatrix[1], " ")
			content[6] = strings.Join(fingermatrix[2], " ")

			b := []byte(strings.Join(content, "\n"))

			err = os.WriteFile(filepath, b, 0644)
			if err != nil {
				self.message("error!", err.Error())
			} else {
				self.message(fmt.Sprintf("saved to %s!", filepath))
			}
		}
	}
}
