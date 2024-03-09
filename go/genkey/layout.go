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
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	websocket "github.com/gorilla/websocket"
)

type GenkeyLayout struct {
	conn     *websocket.Conn
	userData *UserData
}

func NewGenkeyLayout(conn *websocket.Conn, userData *UserData) *GenkeyLayout {
	return &GenkeyLayout{conn, userData}
}

func (self *GenkeyLayout) SendMessage(s string) {
	self.conn.WriteMessage(websocket.TextMessage, []byte(s))
}

type Pos struct {
	Col int
	Row int
}

type Pair [2]Pos
type Finger int

type KeymapMutexMap struct {
	mu   sync.RWMutex
	mmap map[string]Pos
}

func (km *KeymapMutexMap) Get(key string) Pos {
	km.mu.Lock()
	defer km.mu.Unlock()
	return km.mmap[key]
}

func (km *KeymapMutexMap) TryGet(key string) (Pos, bool) {
	km.mu.Lock()
	defer km.mu.Unlock()
	value, ok := km.mmap[key]
	return value, ok
}

func (km *KeymapMutexMap) Store(key string, pos Pos) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.mmap[key] = pos
}

func (km *KeymapMutexMap) Pop(key string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	delete(km.mmap, key)
}

func (km *KeymapMutexMap) Update(newMap map[string]Pos) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.mmap = newMap
}

func (km *KeymapMutexMap) CopyMap() map[string]Pos {
	km.mu.Lock()
	defer km.mu.Unlock()

	newMap := make(map[string]Pos)

	for k, v := range km.mmap {
		newMap[k] = v
	}

	return newMap
}

type Layout struct {
	Name         string
	Keys         [][]string
	Keymap       KeymapMutexMap
	Fingermatrix map[Pos]Finger
	Fingermap    map[Finger][]Pos
	Total        float64
}

func (self *GenkeyLayout) MinimizeLayout(init *Layout, pins [][]string, count int, top bool, is33 bool, noCross bool) {
	genkeyGenerate := NewGenkeyGenerate(self.conn, self.userData)
	genkeyInteractive := NewGenkeyInteractive(self.conn, self.userData)

	bestScore := genkeyGenerate.Score(init)
	bestLayout := genkeyInteractive.CopyLayout(init)
	var tot int
	var r1len int
	var r2len int
	if is33 {
		tot = 33
		r1len = 12
		r2len = 11
	} else {
		tot = 30
		r1len = 10
		r2len = 10
	}
	var foundBetter bool
	for {
		foundBetter = false
		bestSoFarScore := bestScore
		bestSoFarLayout := bestLayout

		for i := 0; i < tot-1; i++ {
			for j := i + 1; j < tot; j++ {
				var irow int
				var icol int
				if i < r1len {
					irow = 0
				} else if i < (r1len + r2len) {
					irow = 1
				} else {
					irow = 2
				}
				if i < r1len {
					icol = i
				} else if i < (r1len + r2len) {
					icol = i - r1len
				} else {
					icol = i - (r1len + r2len)
				}
				var jrow int
				var jcol int
				if j < r1len {
					jrow = 0
				} else if j < (r1len + r2len) {
					jrow = 1
				} else {
					jrow = 2
				}
				if j < r1len {
					jcol = j
				} else if j < (r1len + r2len) {
					jcol = j - r1len
				} else {
					jcol = j - (r1len + r2len)
				}
				if noCross {
					if !((icol <= 4 && jcol <= 4) || (icol >= 5 && jcol >= 5)) {
						continue
					}
				}
				pi := pins[irow][icol]
				pj := pins[jrow][jcol]
				if pi == "#" || pj == "#" {
					continue
				}
				swapped := genkeyInteractive.CopyLayout(bestLayout)
				ki := swapped.Keys[irow][icol]
				kj := swapped.Keys[jrow][jcol]
				if pi == ki || pi == kj || pj == ki || pj == kj {
					continue
				}

				genkeyGenerate.Swap(swapped, swapped.Keymap.Get(ki), swapped.Keymap.Get(kj))

				var swappedScore float64
				if count != 0 {
					self.MinimizeLayout(swapped, pins, count-1, false, is33, noCross)
					recBestScore := genkeyGenerate.Score(swapped)
					if recBestScore < bestSoFarScore {
						bestSoFarScore = recBestScore
						bestSoFarLayout = swapped
						foundBetter = true
					}
				}
				swappedScore = genkeyGenerate.Score(swapped)
				if swappedScore < bestSoFarScore {
					bestSoFarScore = swappedScore
					bestSoFarLayout = swapped
					foundBetter = true
				}
			}
		}
		if bestSoFarScore < bestScore {
			bestScore = bestSoFarScore
			bestLayout = bestSoFarLayout
		}
		if !foundBetter {
			break
		}
	}
	init = bestLayout
}

func (self *GenkeyLayout) LoadLayout(f string) *Layout {
	var l Layout
	b, err := GenkeyReadFile(f)
	if err != nil {
		panic(err)
	}

	s := string(b)
	lines := strings.Split(s, "\n")
	if len(lines) < 7 {
		panic(fmt.Sprintf("WARNING: Layout in file %s is formatted incorrectly, ignoring\n", f))
	}
	l.Name = strings.TrimSpace(lines[0])
	l.Keys = make([][]string, 3)
	keys := lines[1:4]
	for line := range keys {
		separated := true
		for _, rune := range keys[line] {
			c := string(rune)
			c = strings.ToLower(c)
			if c == " " {
				separated = true
				continue
			} else if !separated {
				continue
			} else {
				separated = false
				l.Keys[line] = append(l.Keys[line], c)
				l.Total += float64(self.userData.Data.Letters[c])
			}
		}
	}
	l.Fingermatrix = make(map[Pos]Finger, 3)
	l.Fingermap = make(map[Finger][]Pos)
	for y, row := range lines[4:7] {
		separated := true

		tx := 0
		for _, c := range strings.Split(strings.TrimSpace(row), "") {
			if c == " " {
				separated = true
				continue
			} else if !separated {
				continue
			}
			n, err := strconv.Atoi(c)
			if err != nil {
				panic(fmt.Sprintf("%s layout fingermatrix is badly formatted!\n%v", f, err))
			}
			separated = false
			fg := Finger(n)
			l.Fingermatrix[Pos{tx, y}] = fg
			l.Fingermap[fg] = append(l.Fingermap[fg], Pos{tx, y})
			tx++
		}
	}

	l.Keymap.Update(self.GenKeymap(l.Keys))

	return &l
}

func (self *GenkeyLayout) LoadLayoutDir() {
	dir, err := GenkeyOpen(self.userData.Config.Paths.Layouts)
	if err != nil {
		panic(fmt.Sprintf("Layouts directory could not be opened at %s\n%v", self.userData.Config.Paths.Layouts, err))
	}
	files, _ := dir.Readdirnames(0)
	for _, f := range files {
		l := self.LoadLayout(filepath.Join(self.userData.Config.Paths.Layouts, f))
		if l.Name == "" {
			continue
		}
		if !strings.HasPrefix(f, "_") {
			self.userData.Layouts[strings.ToLower(l.Name)] = l
		} else {
			self.userData.GeneratedFingermap = l.Fingermap
			self.userData.GeneratedFingermatrix = l.Fingermatrix
			for y, row := range l.Keys {
				for x, k := range row {
					if k == "*" {
						self.userData.SwapPossibilities = append(self.userData.SwapPossibilities, Pos{x, y})
					}
				}
			}
		}
	}
}

// func NewLayout(name string, keys string) Layout {
// 	s := strings.Split(keys, "")
// 	return Layout{name, s, GenKeymap(s), FingerMap}
// }

func (self *GenkeyLayout) GenKeymap(keys [][]string) map[string]Pos {
	keymap := make(map[string]Pos)
	for y, row := range keys {
		for x, v := range row {
			keymap[v] = Pos{x, y}
		}
	}
	return keymap
}

func (self *GenkeyLayout) FingerSpeed(l *Layout, weighted bool) []float64 {
	speeds := []float64{0, 0, 0, 0, 0, 0, 0, 0}
	weight := &self.userData.Config.Weights
	sfbweight := weight.FSpeed.SFB
	dsfbweight := weight.FSpeed.DSFB
	for f, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			for j := i; j < len(posits); j++ {
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]

				sfb := float64(self.userData.Data.Bigrams[*k1+*k2])
				dsfb := self.userData.Data.Skipgrams[*k1+*k2]
				if i != j {
					sfb += float64(self.userData.Data.Bigrams[*k2+*k1])
					dsfb += self.userData.Data.Skipgrams[*k2+*k1]
				}

				dist := self.twoKeyDist(*p1, *p2, true) + (2 * weight.FSpeed.KeyTravel)
				speeds[f] += ((sfbweight * sfb) + (dsfbweight * dsfb)) * dist
			}
		}
		if weighted {
			speeds[f] /= weight.FSpeed.KPS[f]
		}
		speeds[f] = 800 * speeds[f] / l.Total
	}
	return speeds
}

func (self *GenkeyLayout) DynamicFingerSpeed(l *Layout, weighted bool) []float64 {
	speeds := []float64{0, 0, 0, 0, 0, 0, 0, 0}
	weight := &self.userData.Config.Weights
	sfbweight := weight.FSpeed.SFB
	dsfbweight := weight.FSpeed.DSFB
	for f, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			var highestsfb float64
			var highestdsfb float64
			var highestdist float64
			var highestspeed float64
			for j := 0; j < len(posits); j++ {
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]

				sfb := float64(self.userData.Data.Bigrams[*k1+*k2])
				dsfb := self.userData.Data.Skipgrams[*k1+*k2]

				dist := self.twoKeyDist(*p1, *p2, true) + (2 * weight.FSpeed.KeyTravel)
				speed := ((sfbweight * sfb) + (dsfbweight * dsfb)) * dist
				if sfb > highestsfb {
					highestsfb = sfb
					highestdsfb = dsfb
					highestdist = dist
					highestspeed = speed
				}
				speeds[f] += speed
			}
			newspeed := (dsfbweight * highestdsfb) * highestdist
			speeds[f] -= highestspeed
			speeds[f] += newspeed
		}
		if weighted {
			speeds[f] /= weight.FSpeed.KPS[f]
		}
		speeds[f] = 800 * speeds[f] / l.Total
	}
	return speeds
}

func (self *GenkeyLayout) SFBs(l *Layout, skipgrams bool) float64 {
	var count float64
	for _, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			for j := i; j < len(posits); j++ {
				if i == j {
					continue
				}
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]
				if !skipgrams {
					count += float64(self.userData.Data.Bigrams[*k1+*k2] + self.userData.Data.Bigrams[*k2+*k1])
				} else {
					count += self.userData.Data.Skipgrams[*k1+*k2] + self.userData.Data.Skipgrams[*k2+*k1]
				}
			}
		}
	}
	return count
}

func (self *GenkeyLayout) DynamicSFBs(l *Layout) float64 {
	var count float64
	for _, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			var highest float64
			for j := 0; j < len(posits); j++ {
				if i == j {
					continue
				}
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]
				sfb := float64(self.userData.Data.Bigrams[*k1+*k2])
				if sfb > highest {
					highest = sfb
				}
				count += sfb
			}
			count -= highest
		}
	}
	return count
}

type FreqPair struct {
	Ngram string
	Count float64
}

func (self *GenkeyLayout) SortFreqList(pairs []FreqPair) {
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Count > pairs[j].Count
	})
}

func (self *GenkeyLayout) ListSFBs(l *Layout, skipgrams bool) []FreqPair {
	var list []FreqPair
	for _, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			// since this is output, reversed sfbs cannot
			// be shortcut, so we iterate through all
			// combinations without mirroring (j starts at
			// 0 instead of i)
			for j := 0; j < len(posits); j++ {
				if i == j {
					continue
				}
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]
				var count float64
				ngram := *k1 + *k2
				if !skipgrams {
					count = float64(self.userData.Data.Bigrams[ngram])
				} else {
					count = self.userData.Data.Skipgrams[ngram]
				}
				list = append(list, FreqPair{ngram, count})
			}
		}
	}

	return list
}

func (self *GenkeyLayout) ListDynamic(l *Layout) ([]FreqPair, []FreqPair) {
	sfbs := self.ListSFBs(l, false)
	self.SortFreqList(sfbs)
	var escaped []FreqPair
	var real []FreqPair
	highestfound := make(map[Pos]bool)
	for _, bg := range sfbs {
		prefix := l.Keymap.Get(string(bg.Ngram[0]))
		if highestfound[prefix] {
			real = append(real, bg)
		} else {
			escaped = append(escaped, bg)
			highestfound[prefix] = true
		}
	}

	return escaped, real
}

func (self *GenkeyLayout) ListWorstBigrams(l *Layout) []FreqPair {
	var bigrams []FreqPair
	weight := self.userData.Config.Weights
	sfbweight := weight.FSpeed.SFB
	dsfbweight := weight.FSpeed.DSFB
	for f, posits := range l.Fingermap {
		for i := 0; i < len(posits); i++ {
			for j := i; j < len(posits); j++ {
				p1 := &posits[i]
				p2 := &posits[j]
				k1 := &l.Keys[p1.Row][p1.Col]
				k2 := &l.Keys[p2.Row][p2.Col]
				sfb := float64(self.userData.Data.Bigrams[*k1+*k2])
				dsfb := self.userData.Data.Skipgrams[*k1+*k2]
				if i != j {
					sfb += float64(self.userData.Data.Bigrams[*k2+*k1])
					dsfb += self.userData.Data.Skipgrams[*k2+*k1]
				}

				dist := self.twoKeyDist(*p1, *p2, true) + (2 * weight.FSpeed.KeyTravel)
				cost := 100 * (((sfbweight * sfb) + (dsfbweight * dsfb)) * dist) / weight.FSpeed.KPS[f]
				bigrams = append(bigrams, FreqPair{*k1 + *k2, cost})
			}
		}
	}
	return bigrams
}

type TrigramValues struct {
	RightInwardRolls  int
	RightOutwardRolls int
	LeftInwardRolls   int
	LeftOutwardRolls  int
	Alternates        int
	Onehands          int
	Redirects         int
	Total             int
}

// FastTrigrams approximates trigram counts with a given precision
// (precision=0 gives full data). It returns a count of {rolls,
// alternates, onehands, redirects, total}
func (self *GenkeyLayout) FastTrigrams(l *Layout, precision int) TrigramValues {
	var tgs TrigramValues

	if precision == 0 {
		precision = len(self.userData.Data.TopTrigrams)
	}

	for _, tg := range self.userData.Data.TopTrigrams[:min(len(self.userData.Data.TopTrigrams), precision)] {
		km1, ok1 := l.Keymap.TryGet(string(tg.Ngram[0]))
		km2, ok2 := l.Keymap.TryGet(string(tg.Ngram[1]))
		km3, ok3 := l.Keymap.TryGet(string(tg.Ngram[2]))

		if !ok1 || !ok2 || !ok3 {
			continue
		}

		f1 := l.Fingermatrix[km1]
		f2 := l.Fingermatrix[km2]
		f3 := l.Fingermatrix[km3]

		tgs.Total += int(tg.Count)

		if f1 != f2 && f2 != f3 {
			h1 := (f1 >= 4)
			h2 := (f2 >= 4)
			h3 := (f3 >= 4)

			if h1 == h2 && h2 == h3 {
				dir1 := f1 < f2
				dir2 := f2 < f3

				if dir1 == dir2 {
					tgs.Onehands += int(tg.Count)
				} else {
					tgs.Redirects += int(tg.Count)
				}
			} else if h1 != h2 && h2 != h3 {
				tgs.Alternates += int(tg.Count)
			} else {
				rollhand := h2
				rollfirst := (h1 == rollhand)
				var first Finger
				var second Finger
				if rollfirst {
					first = f1
					second = f2
				} else {
					first = f2
					second = f3
				}
				if rollhand == false { // left hand
					if first < second { // inward roll
						tgs.LeftInwardRolls += int(tg.Count)
						//println("Left Inward Roll: ", tg.Ngram)
					} else {
						tgs.LeftOutwardRolls += int(tg.Count)
						//println("Left Outward Roll: ", tg.Ngram)
					}
				} else if rollhand == true { // right hand
					if first > second { // inward roll
						tgs.RightInwardRolls += int(tg.Count)
						//println("Right Inward Roll: ", tg.Ngram)
					} else {
						tgs.RightOutwardRolls += int(tg.Count)
						//println("Right Outward Roll:", tg.Ngram)
					}
				}
			}
		}
	}

	return tgs
}

func (self *GenkeyLayout) IndexUsage(l *Layout) (float64, float64) {
	left := 0
	right := 0

	for _, pos := range l.Fingermap[3] {
		key := l.Keys[pos.Row][pos.Col]
		left += self.userData.Data.Letters[key]
	}
	for _, pos := range l.Fingermap[4] {
		key := l.Keys[pos.Row][pos.Col]
		right += self.userData.Data.Letters[key]
	}

	return (100 * float64(left) / l.Total), (100 * float64(right) / l.Total)
}

func (self *GenkeyLayout) LSBs(l *Layout) int {
	var count int

	// LI LM
	for _, p1 := range l.Fingermap[3] {
		for _, p2 := range l.Fingermap[2] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				count += self.userData.Data.Bigrams[k1+k2]
				count += self.userData.Data.Bigrams[k2+k1]
			}
		}
	}

	// RI RM
	for _, p1 := range l.Fingermap[4] {
		for _, p2 := range l.Fingermap[5] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				count += self.userData.Data.Bigrams[k1+k2]
				count += self.userData.Data.Bigrams[k2+k1]
			}
		}
	}

	// LP LR
	for _, p1 := range l.Fingermap[0] {
		for _, p2 := range l.Fingermap[1] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				count += self.userData.Data.Bigrams[k1+k2]
				count += self.userData.Data.Bigrams[k2+k1]
			}
		}
	}

	// RP RR
	for _, p1 := range l.Fingermap[7] {
		for _, p2 := range l.Fingermap[6] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				count += self.userData.Data.Bigrams[k1+k2]
				count += self.userData.Data.Bigrams[k2+k1]
			}
		}
	}
	return count
}

func (self *GenkeyLayout) ListLSBs(l *Layout) []FreqPair {
	var list []FreqPair
	for _, p1 := range l.Fingermap[3] {
		for _, p2 := range l.Fingermap[2] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				list = append(list, FreqPair{k1 + k2, float64(self.userData.Data.Bigrams[k1+k2])})
				list = append(list, FreqPair{k2 + k1, float64(self.userData.Data.Bigrams[k2+k1])})
			}
		}
	}

	for _, p1 := range l.Fingermap[4] {
		for _, p2 := range l.Fingermap[5] {
			var dist float64
			if self.userData.StaggerFlag {
				dist = math.Abs(self.staggeredX(p1.Col, p1.Row) - self.staggeredX(p2.Col, p2.Row))
			} else {
				dist = math.Abs(float64(p1.Col - p2.Col))
			}
			if dist >= 2 {
				k1 := l.Keys[p1.Row][p1.Col]
				k2 := l.Keys[p2.Row][p2.Col]
				list = append(list, FreqPair{k1 + k2, float64(self.userData.Data.Bigrams[k1+k2])})
				list = append(list, FreqPair{k2 + k1, float64(self.userData.Data.Bigrams[k2+k1])})
			}
		}
	}
	return list
}

func (self *GenkeyLayout) ColRow(pos int) (int, int) {
	var col int
	var row int
	if pos < 10 {
		col = pos
		row = 0
	} else if pos < 20 {
		col = pos - 10
		row = 1
	} else if pos < 30 {
		col = pos - 20
		row = 2
	}

	return col, row
}

func (self *GenkeyLayout) Similarity(a, b []string) int {
	var score int
	for i := 0; i < 30; i++ {
		weight := 1
		if i >= 10 && i <= 13 {
			weight = 2
		} else if i >= 16 && i <= 19 {
			weight = 2
		}
		if a[i] == b[i] {
			score += weight
		}
	}
	return score
}

func (self *GenkeyLayout) DuplicatesAndMissing(l *Layout) ([]string, []string) {
	counts := make(map[string]int)
	// collect counts of each key
	for _, row := range l.Keys {
		for _, c := range row {
			counts[c] += 1
		}
	}
	// then check duplicates and missing
	duplicates := make([]string, 0)
	missing := make([]string, 0)
	for _, r := range []rune("abcdefghijklmnopqrstuvwxyz,./;'") {
		c := string(r)
		if counts[c] == 0 {
			missing = append(missing, c)
		} else if counts[c] > 1 {
			duplicates = append(duplicates, c)
		}
	}
	return duplicates, missing
}

func (self *GenkeyLayout) staggeredX(c, r int) float64 {
	var sx float64
	if r == 0 {
		sx = float64(c) - 0.25
	} else if r == 2 {
		sx = float64(c) + 0.5
	} else {
		sx = float64(c)
	}
	return sx
}

func (self *GenkeyLayout) staggeredY(c, r int) float64 {
	config := &self.userData.Config
	var sy float64
	if c < 10 {
		sy = float64(r) - config.Weights.ColStaggers[c]
	} else if c >= 10 {
		// Unsure if pinky stagger being the same is guaranteed
		sy = float64(r) - config.Weights.ColStaggers[9]
	}
	return sy
}

func (self *GenkeyLayout) twoKeyDist(a, b Pos, weighted bool) float64 {
	var ax float64
	var bx float64
	var ay float64
	var by float64
	userData := self.userData

	if self.userData.StaggerFlag {
		ax = self.staggeredX(a.Col, a.Row)
		bx = self.staggeredX(b.Col, b.Row)
	} else {
		ax = float64(a.Col)
		bx = float64(b.Col)
	}

	if userData.ColStaggerFlag {
		ay = self.staggeredY(a.Col, a.Row)
		by = self.staggeredY(b.Col, b.Row)
	} else {
		ay = float64(a.Row)
		by = float64(b.Row)
	}

	x := ax - bx
	y := ay - by

	var dist float64
	if weighted {
		dist = (self.userData.Config.Weights.Dist.Lateral * x * x) + (y * y)
	} else {
		dist = math.Sqrt((x * x) + (y * y))
	}
	return dist
}
