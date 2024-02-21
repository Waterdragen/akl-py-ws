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
	"os"
	"path/filepath"

	util "github.com/waterdragen/akl-ws/util"
)

var FingerNames = [8]string{"LP", "LR", "LM", "LI", "RI", "RM", "RR", "RP"}

type UserConfig struct {
	Corpus string
	Output struct {
		Generation struct {
			Heatmap bool
		}
		Rank struct {
			Spacer string
		}
		Analysis struct {
			TopNgrams int
		}
		Misc struct {
			TopNgrams int
		}
	}
	Paths struct {
		Layouts string
		Corpora string
		Heatmap string
	}
	Weights struct {
		Stagger     bool
		ColStagger  bool
		ColStaggers [10]float64
		FSpeed      struct {
			SFB       float64
			DSFB      float64
			KeyTravel float64
			KPS       [8]float64
		}
		Dist struct {
			Lateral float64
		}
		Score struct {
			FSpeed       float64
			IndexBalance float64
			LSB          float64

			Trigrams struct {
				Enabled          bool
				Precision        int
				LeftInwardRoll   float64
				LeftOutwardRoll  float64
				RightInwardRoll  float64
				RightOutwardRoll float64
				Alternate        float64
				Redirect         float64
				Onehand          float64
			}
		}
	}
	Generation struct {
		GeneratedLayoutChars string
		InitialPopulation    int
		Selection            int
	}
	CorpusProcessing struct {
		ValidChars                  string
		CharSubstitutions           [][2]string
		MaxSkipgramSize             int8
		SkipgramsMustSpanValidChars bool
	}
}

type UserInteractive struct {
	Aswaps        []Pos
	Bswaps        []Pos
	Swapnum       int
	Pins          [][]string
	Threshold     float64
	InInteractive bool
	Layout        Layout
	LayoutWidth   int
}

type UserData struct {
	// From globals.go
	StaggerFlag    bool
	ColStaggerFlag bool
	ColStaggers    [10]float64
	SlideFlag      bool
	DynamicFlag    bool
	ImproveFlag    bool
	ImproveLayout  Layout

	Layouts               map[string]Layout
	GeneratedFingermap    map[Finger][]Pos
	GeneratedFingermatrix map[Pos]Finger
	LongestLayoutName     int

	SwapPossibilities []Pos
	Analyzed          int

	// From main.go
	Data TextData

	// From generate.go
	GoroutineCounter util.AtomicCounter

	// other
	Config      UserConfig
	Interactive UserInteractive // interactive.go
}

const importerToGenkey = "./genkey"

func GenkeyOpen(path string) (*os.File, error) {
	return os.Open(filepath.Join(importerToGenkey, path))
}

func GenkeyReadFile(path string) ([]byte, error) {
	return os.ReadFile(filepath.Join(importerToGenkey, path))
}
