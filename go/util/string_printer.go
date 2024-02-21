package util

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/wayneashleyberry/truecolor/pkg/color"
)

type AnsiByte struct {
	EscapeSequence *string
	Char           byte
}

type StringPrinter struct {
	Width  int
	Height int

	// Private
	console   [][]AnsiByte
	cursorRow int
	cursorCol int
}

func NewStringPrinter() *StringPrinter {
	sp := StringPrinter{Height: 60, Width: 120}
	sp.newConsole()
	return &sp
}

func (sp *StringPrinter) newConsole() {
	if sp.Width == 0 || sp.Height == 0 {
		panic(fmt.Sprintf("Console width or height cannot be 0, got: (%d, %d)", sp.Width, sp.Height))
	}

	sp.cursorRow = 0
	sp.cursorCol = 0

	sp.console = make([][]AnsiByte, sp.Height)
	for r := 0; r < sp.Height; r++ {
		sp.console[r] = make([]AnsiByte, sp.Width)
		for c := 0; c < sp.Width; c++ {
			sp.console[r][c] = AnsiByte{nil, ' '}
		}
	}
}

func (sp *StringPrinter) flushConsole() {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			sp.console[r][c] = AnsiByte{nil, ' '}
		}
	}
}

func (sp *StringPrinter) PrintColor(c *color.Message, s string) {
	if sp.console == nil {
		panic("Did not instantiate sp.console")
	}

	sb := []byte(s)

	for _, b := range sb {
		// Skip a line instead of writing character to array
		if b == '\n' || b == '\r' {
			sp.cursorRow++
			sp.cursorCol = 0
			continue
		}

		if sp.cursorCol >= sp.Width {
			sp.cursorRow++
			sp.cursorCol = 0
		}

		if c != nil {
			colorByte := c.Sprint(string(b))
			sp.console[sp.cursorRow][sp.cursorCol] = AnsiByte{&colorByte, b}
		} else {
			sp.console[sp.cursorRow][sp.cursorCol] = AnsiByte{nil, b}
		}
		sp.cursorCol++
	}
}

func (sp *StringPrinter) Print(s string) {
	sp.PrintColor(nil, s)
}

func (sp *StringPrinter) Flush() string {
	stringBuilder := strings.Builder{}

	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			ansiByte := &sp.console[r][c]
			if ansiByte.EscapeSequence == nil {
				stringBuilder.WriteByte(ansiByte.Char)
			} else {
				s := *ansiByte.EscapeSequence
				stringBuilder.WriteString(s)
			}
		}
		stringBuilder.WriteByte('\n')
	}

	consoleStr := stringBuilder.String()
	sp.flushConsole()
	return consoleStr
}

func (sp *StringPrinter) FlushAndTrim() string {
	s := sp.Flush()
	invisChars := regexp.MustCompile(`[\s\t\n\r]*$`)
	return invisChars.ReplaceAllString(s, "")
}

func (sp *StringPrinter) Clear() {
	sp.newConsole()
}

func (sp *StringPrinter) MoveCursor(x int, y int) {
	sp.cursorRow = y
	sp.cursorCol = x
}

func (sp *StringPrinter) MoveCursorUp(bias int) {
	sp.cursorRow -= bias
}

func (sp *StringPrinter) MoveCursorDown(bias int) {
	sp.cursorRow += bias
}

func (sp *StringPrinter) MoveCursorForward(bias int) {
	sp.cursorCol += bias
}

func (sp *StringPrinter) MoveCursorBackward(bias int) {
	sp.cursorCol -= bias
}
