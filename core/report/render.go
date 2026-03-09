package report

import (
	"fmt"
	"math"
	"strings"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/tik"
)

type LineInfo struct {
	Line    int
	Col     int
	Content string
}

type Renderer struct {
	gutterWidth int
}

type renderLine struct {
	LineNum int
	Content string
	IsSrc   bool
}

func (r *Renderer) Render(report Report) string {
	var sb strings.Builder

	// header
	fmt.Fprintf(&sb, "%s[%04d]: %s\n", report.Kind, report.Code, report.Message)

	if len(report.Labels) == 0 {
		return sb.String()
	}

	source := report.Labels[0].Source
	labelsRange := tik.TextRange{
		Start: math.MaxInt,
		End:   math.MinInt,
	}
	for _, label := range report.Labels {
		labelsRange.Start = min(labelsRange.Start, label.Range.Start)
		labelsRange.End = max(labelsRange.End, label.Range.End)
		source = label.Source
	}

	srcLines := r.getLinesInRange(source, labelsRange)
	if len(srcLines) == 0 {
		return sb.String()
	}

	// prepare annotations
	annotations := make(map[int][]string)
	for _, label := range report.Labels {
		info := rangeToLineInfo(source, label.Range)
		caret := strings.Repeat(" ", info.Col) + "^"
		length := label.Range.End - label.Range.Start
		if length > 1 {
			caret += strings.Repeat("^", length-1)
		}
		if label.Note != "" {
			caret += " " + label.Note
		}
		annotations[info.Line] = append(annotations[info.Line], caret)
	}

	// build canvas
	var canvas []renderLine
	for _, li := range srcLines {
		canvas = append(canvas, renderLine{LineNum: li.Line, Content: li.Content, IsSrc: true})
		if notes, ok := annotations[li.Line]; ok {
			for _, note := range notes {
				canvas = append(canvas, renderLine{LineNum: 0, Content: note, IsSrc: false})
			}
		}
	}

	// calculate gutter
	maxLine := srcLines[len(srcLines)-1].Line
	r.gutterWidth = len(fmt.Sprintf("%d", maxLine)) + 1

	// render
	if report.Location.FileName != "" {
		fmt.Fprintf(&sb, "%s ┌─ %s:%d:%d\n", r.inGutter(""), report.Location.FileName, report.Location.Line, report.Location.Col)
	}
	fmt.Fprintf(&sb, "%s │\n", r.inGutter(""))

	for _, row := range canvas {
		if row.IsSrc {
			fmt.Fprintf(&sb, "%s │ ", r.inGutter(fmt.Sprint(row.LineNum)))
			syntaxHighlight(&sb, row)
			fmt.Fprint(&sb, "\n")
		} else {
			fmt.Fprintf(&sb, "%s │ %s\n", r.inGutter(""), row.Content)
		}
	}

	// global notes
	if len(report.Notes) > 0 {
		fmt.Fprintf(&sb, "%s │\n", r.inGutter(""))
		for _, note := range report.Notes {
			fmt.Fprintf(&sb, "%s = note: %s\n", r.inGutter(""), note)
		}
	}

	return sb.String()
}

func (r *Renderer) inGutter(s string) string {
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
				Line:    currentLine,
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

func rangeToLineInfo(src luther.SourceCode, tr tik.TextRange) LineInfo {
	currentLine := 1
	lineStartOffset := 0

	lines := strings.Split(string(src.Raw), "\n")

	for _, content := range lines {
		lineEndOffset := lineStartOffset + len([]rune(content))

		// Check if the start of the range falls within the current line's character offsets.
		if tr.Start >= lineStartOffset && tr.Start <= lineEndOffset {
			col := tr.Start - lineStartOffset
			return LineInfo{
				Line:    currentLine,
				Content: content,
				Col:     col,
			}
		}

		lineStartOffset = lineEndOffset + 1 // +1 for the newline character
		currentLine++
	}

	// This would happen if the text range is out of bounds of the source.
	return LineInfo{}
}
