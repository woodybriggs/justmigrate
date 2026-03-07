package report

import (
	"fmt"
	"strings"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/tik"
)

type Label struct {
	Source luther.SourceCode
	Range  tik.TextRange
	Note   string
}

func (label Label) String() string {
	return fmt.Sprintf("%s:%d:%d %s", label.Source.FileName, label.Range.Start, label.Range.End, label.Note)
}

type Report struct {
	Kind    string
	Code    int
	Message string
	Labels  []Label
	Notes   []string
}

func (r *Report) Error() string {
	renderer := Renderer{}
	return renderer.Render(*r)
}

type LineInfo struct {
	LineNum int
	Content string
	Col     int
}

type Renderer struct {
	gutterWidth int
}

func (r *Renderer) Render(report Report) string {
	var sb strings.Builder

	// header
	fmt.Fprintf(&sb, "%s[%04d]: %s\n", report.Kind, report.Code, report.Message)

	// labels
	for i, label := range report.Labels {
		lines := r.getLinesInRange(label.Source, label.Range)
		if len(lines) == 0 {
			continue
		}

		// calculate gutter based on the highest line number in this snippet
		maxLine := lines[len(lines)-1].LineNum
		r.gutterWidth = len(fmt.Sprintf("%d", maxLine)) + 1

		// snippet Header
		fmt.Fprintf(&sb, "%s ┌─ %s:%d:%d\n", r.pad(""), label.Source.FileName, lines[0].LineNum, lines[0].Col)
		fmt.Fprintf(&sb, "%s │\n", r.pad(""))

		// source Lines
		for _, li := range lines {
			fmt.Fprintf(&sb, "%s │ %s\n", r.pad(fmt.Sprint(li.LineNum)), li.Content)
		}

		// arrows / underline
		// one would track vertical connectors here.
		firstLine := lines[0]
		pointer := r.pad("") + " │ " + strings.Repeat(" ", firstLine.Col) + "^"

		// if it's a short range on one line, add more carets
		if len(lines) == 1 && label.Range.End-label.Range.Start > 1 {
			pointer += strings.Repeat("^", (label.Range.End-label.Range.Start)-1)
		}

		if label.Note != "" {
			pointer += " " + label.Note
		}
		sb.WriteString(pointer + "\n")

		if i < len(report.Labels)-1 {
			fmt.Fprintf(&sb, "%s │\n", r.pad(""))
		}
	}

	// global notes
	if len(report.Notes) > 0 {
		fmt.Fprintf(&sb, "%s │\n", r.pad(""))
		for _, note := range report.Notes {
			fmt.Fprintf(&sb, "%s = note: %s\n", r.pad(""), note)
		}
	}

	return sb.String()
}

func (r *Renderer) pad(s string) string {
	return fmt.Sprintf("%*s", r.gutterWidth, s)
}

// getLinesInRange converts the flat Raw rune slice into a slice of LineInfo
func (r *Renderer) getLinesInRange(src luther.SourceCode, tr tik.TextRange) []LineInfo {
	var result []LineInfo

	currentLine := 1
	lineStartOffset := 0

	lines := strings.Split(string(src.Raw), "\n")

	for _, content := range lines {
		lineEndOffset := lineStartOffset + len([]rune(content))

		if lineEndOffset >= tr.Start && lineStartOffset <= tr.End {
			col := 0
			if tr.Start > lineStartOffset {
				col = tr.Start - lineStartOffset
			}

			result = append(result, LineInfo{
				LineNum: currentLine,
				Content: content,
				Col:     col,
			})
		}

		lineStartOffset = lineEndOffset + 1 // +1 for the \n
		currentLine++

		if lineStartOffset > tr.End {
			break
		}
	}
	return result
}

func NewReport(kind string) *Report {
	return &Report{
		Kind: kind,
	}
}

func (report *Report) WithCode(code int) *Report {
	report.Code = code
	return report
}

func (report *Report) WithMessage(message string) *Report {
	report.Message = message
	return report
}

func (report *Report) WithLabels(labels []Label) *Report {
	report.Labels = append(report.Labels, labels...)
	return report
}

func (report *Report) WithNotes(notes []string) *Report {
	report.Notes = append(report.Notes, notes...)
	return report
}
