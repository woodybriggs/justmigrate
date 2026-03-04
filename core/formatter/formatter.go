package formatter

import (
	"fmt"
	"io"
	"strings"
	"woodybriggs/justmigrate/datastructures"
)

type Formatter interface {
	Identifier(s string)
	Text(s string)
	Rune(r rune)
	Space()
	Line()            // "soft" line (space if fits, newline if broken)
	Break()           // "hard" newline
	Indent(fn func()) // increase logical nesting
	Anchor(fn func()) // align subsequent lines to current column
	Group(fn func())  // try to fit everything inside on one line
}

type FormatMode int

const (
	FormatModeRender  FormatMode = 1
	FormatModeMeasure FormatMode = 2
)

type CoreFormatter struct {
	writer                io.Writer
	maxWidth              int
	indentStr             string
	escapeIdentifierStart string
	escapeIdentifierEnd   string

	// State
	column      int
	indentLevel int
	anchorStack datastructures.Stack[int]

	// Grouping/Recording state
	isRecording  bool
	recordBuffer []func() // Sequence of operations to replay

	mode        FormatMode
	groupBroken bool
}

func NewCoreFormatter(w io.Writer, maxWidth int, escapeIdentWith string) *CoreFormatter {
	var escapeIdentifierStart string = ""
	var escapeIdentifierEnd string = ""
	if len(escapeIdentWith) == 2 {
		escapeIdentifierStart = string(escapeIdentWith[0])
		escapeIdentifierEnd = string(escapeIdentWith[1])
	}
	return &CoreFormatter{
		writer:                w,
		maxWidth:              maxWidth,
		indentStr:             "    ",
		mode:                  FormatModeRender,
		groupBroken:           false,
		escapeIdentifierStart: escapeIdentifierStart,
		escapeIdentifierEnd:   escapeIdentifierEnd,
	}
}

func (f *CoreFormatter) Space() {
	f.Rune(' ')
}

func (f *CoreFormatter) writeIndent() {
	if f.column == 0 {
		padding := f.getPadding()
		// Use spaces for alignment/anchors, or f.indentStr for tabs
		f.writer.Write([]byte(strings.Repeat(" ", padding)))
		f.column = padding
	}
}

func (f *CoreFormatter) Identifier(s string) {
	f.Text(f.escapeIdentifierStart)
	f.Text(s)
	f.Text(f.escapeIdentifierEnd)
}

func (f *CoreFormatter) Text(s string) {
	if f.mode == FormatModeMeasure {
		f.column += len(s)
		return
	}
	f.writeIndent()
	f.writer.Write([]byte(s))
	f.column += len(s)
}

func (f *CoreFormatter) Rune(r rune) {
	if f.mode == FormatModeMeasure {
		f.column++
		return
	}
	f.writeIndent()
	fmt.Fprintf(f.writer, "%c", r)
	f.column++
}

func (f *CoreFormatter) Line() {
	if f.mode == FormatModeMeasure {
		f.column++ // Measure as a space
		return
	}

	// If the current group fits, Line is a Space.
	// If the group is "broken" (too long), Line is a Break.
	if !f.groupBroken {
		f.Space()
	} else {
		f.Break()
	}
}

func (f *CoreFormatter) Break() {
	if f.mode == FormatModeMeasure {
		// A hard break resets the column during measurement
		f.column = 0
		return
	}

	f.writer.Write([]byte("\n"))
	f.column = 0
}

func (f *CoreFormatter) Indent(fn func()) {
	f.indentLevel++
	fn()
	f.indentLevel--
}

func (f *CoreFormatter) Anchor(fn func()) {
	if f.column == 0 {
		f.anchorStack.Push(f.getPadding())
	} else {
		f.anchorStack.Push(f.column)
	}
	fn()
	f.anchorStack.Pop()
}

func (f *CoreFormatter) Group(fn func()) {

	prevMode := f.mode
	prevColumn := f.column
	prevBroken := f.groupBroken

	f.mode = FormatModeMeasure
	f.groupBroken = false
	fn()

	willFit := f.column <= f.maxWidth

	f.mode = prevMode
	f.column = prevColumn
	f.groupBroken = !willFit

	if f.mode == FormatModeRender {
		fn()
	}

	f.groupBroken = prevBroken
}

func (f *CoreFormatter) getPadding() int {
	// If anchored, use anchor. If not, use indent level.
	if len(f.anchorStack.Data) > 0 {
		v, _ := f.anchorStack.Top()
		return v
	}
	return f.indentLevel * len(f.indentStr)
}
