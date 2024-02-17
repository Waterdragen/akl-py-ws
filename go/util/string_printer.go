package util

import (
	"bytes"
	"fmt"
	"regexp"
)

type StringPrinter struct {
	Width  int
	Height int

	// Private
	console   [][]byte
	cursorRow int
	cursorCol int
}

func NewStringPrinter() *StringPrinter {
	sp := StringPrinter{Height: 30, Width: 120}
	sp.newConsole()
	return &sp
}

func (sp *StringPrinter) newConsole() {
	if sp.Width == 0 || sp.Height == 0 {
		panic(fmt.Sprintf("Console width or height cannot be 0, got: (%d, %d)", sp.Width, sp.Height))
	}

	sp.cursorRow = 0
	sp.cursorCol = 0

	sp.console = make([][]byte, sp.Height)
	for r := 0; r < sp.Height; r++ {
		sp.console[r] = make([]byte, sp.Width)
		for c := 0; c < sp.Width; c++ {
			sp.console[r][c] = ' '
		}
	}
}

func (sp *StringPrinter) flushConsole() {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			sp.console[r][c] = ' '
		}
	}
}

func (sp *StringPrinter) Print(s string) {
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

		// Write a character to array and move cursor forward
		sp.console[sp.cursorRow][sp.cursorCol] = b
		sp.cursorCol++
	}
}

func (sp *StringPrinter) Flush() string {
	joinedLines := bytes.Join(sp.console, []byte("\n"))
	consoleStr := string(joinedLines)
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

func (sp *StringPrinter) MoveCursor(row int, col int) {
	sp.cursorRow = row
	sp.cursorCol = col
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
