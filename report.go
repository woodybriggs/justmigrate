package main

import (
	"fmt"
	"strings"
)

type Label struct {
	Source SourceCode
	Range  TextRange
	Note   string
}

type ErrorReport struct {
	Code    int
	Message string
	Labels  []Label
	Notes   []string
}

// LineInfo helps map offsets to displayable coordinates
type LineInfo struct {
	LineNum int
	Content string
	Col     int
}

// --- Renderer Logic ---

type Renderer struct {
	gutterWidth int
}

func (r *Renderer) Render(report *ErrorReport) string {
	var sb strings.Builder

	// 1. Header
	sb.WriteString(fmt.Sprintf("error[%04d]: %s\n", report.Code, report.Message))

	for i, label := range report.Labels {
		lines := r.getLinesInRange(label.Source, label.Range)
		if len(lines) == 0 {
			continue
		}

		// Calculate gutter based on the highest line number in this snippet
		maxLine := lines[len(lines)-1].LineNum
		r.gutterWidth = len(fmt.Sprintf("%d", maxLine)) + 1

		// 2. Snippet Header
		sb.WriteString(fmt.Sprintf("%s ┌─ %s:%d:%d\n", r.pad(""), label.Source.FileName, lines[0].LineNum, lines[0].Col))
		sb.WriteString(fmt.Sprintf("%s │\n", r.pad("")))

		// 3. Source Lines
		for _, li := range lines {
			sb.WriteString(fmt.Sprintf("%s │ %s\n", r.pad(fmt.Sprint(li.LineNum)), li.Content))
		}

		// 4. Pointer / Underline (Simplified for single-line or start-of-range focus)
		// For a full implementation like the diagram, one would track vertical connectors here.
		firstLine := lines[0]
		pointer := r.pad("") + " │ " + strings.Repeat(" ", firstLine.Col) + "^"

		// If it's a short range on one line, add more carets
		if len(lines) == 1 && label.Range.End-label.Range.Start > 1 {
			pointer += strings.Repeat("^", (label.Range.End-label.Range.Start)-1)
		}

		if label.Note != "" {
			pointer += " " + label.Note
		}
		sb.WriteString(pointer + "\n")

		if i < len(report.Labels)-1 {
			sb.WriteString(fmt.Sprintf("%s ·\n", r.pad("")))
		}
	}

	// 5. Global Notes
	if len(report.Notes) > 0 {
		sb.WriteString(fmt.Sprintf("%s │\n", r.pad("")))
		for _, note := range report.Notes {
			sb.WriteString(fmt.Sprintf("%s = note: %s\n", r.pad(""), note))
		}
	}

	return sb.String()
}

func (r *Renderer) pad(s string) string {
	return fmt.Sprintf("%*s", r.gutterWidth, s)
}

// getLinesInRange converts the flat Raw rune slice into a slice of LineInfo
func (r *Renderer) getLinesInRange(src SourceCode, tr TextRange) []LineInfo {
	var result []LineInfo

	currentLine := 1
	lineStartOffset := 0

	// Identify all lines in the file
	lines := strings.Split(string(src.Raw), "\n")

	for _, content := range lines {
		lineEndOffset := lineStartOffset + len([]rune(content))

		// Check if this line intersects with our TextRange
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

// --- Implementation of your API Methods ---

func NewErrorReport() *ErrorReport {
	return &ErrorReport{}
}

func (report *ErrorReport) WithCode(code int) *ErrorReport {
	report.Code = code
	return report
}

func (report *ErrorReport) WithMessage(message string) *ErrorReport {
	report.Message = message
	return report
}

func (report *ErrorReport) WithLabels(labels []Label) *ErrorReport {
	report.Labels = append(report.Labels, labels...)
	return report
}

func (report *ErrorReport) WithNotes(notes []string) *ErrorReport {
	report.Notes = append(report.Notes, notes...)
	return report
}

// func main() {
// 	rawSource := []rune("package main\n\nfunc main() {\n    println(\"hello world\")\n}")
// 	src := SourceCode{FileName: "main.go", Raw: rawSource}

// 	// Let's target the 'println' call
// 	// "println" starts at index 32
// 	report := NewErrorReport().
// 		WithCode(101).
// 		WithMessage("unresolved reference").
// 		WithLabels([]Label{
// 			{Source: src, Range: TextRange{Start: 32, End: 39}, Note: "did you mean 'fmt.Println'?"},
// 		}).
// 		WithNotes([]string{"import 'fmt' at the top of the file"})

// 	renderer := Renderer{}
// 	fmt.Println(renderer.Render(report))
// }
