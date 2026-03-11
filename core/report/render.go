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
	var out strings.Builder
	var tmp strings.Builder
	// header
	fmt.Fprintf(&out, " ┌─ %s\n", report.Kind)
	fmt.Fprintf(&out, " │\n")

	// labels can be from different sources, so we group
	// labels by source and then find the source range.
	type SourceCodeFileName string
	sources := map[SourceCodeFileName]*luther.SourceCode{}
	orderSource := []SourceCodeFileName{}
	sourceLabels := map[SourceCodeFileName][]*Label{}
	sourceLines := map[SourceCodeFileName][]LineInfo{}

	// group the labels up
	for i := range report.Labels {
		filename := SourceCodeFileName(report.Labels[i].Source.FileName)
		if _, ok := sources[filename]; !ok {
			orderSource = append(orderSource, filename)
			sources[filename] = &report.Labels[i].Source
		}
		sourceLabels[filename] = append(sourceLabels[filename], &report.Labels[i])
	}

	// get the source lines from the labels
	for sourceFileName, labels := range sourceLabels {
		start := math.MaxInt
		end := math.MinInt

		for _, label := range labels {
			start = min(start, label.Range.Start)
			end = max(end, label.Range.End)
		}

		sourceLines[sourceFileName] = r.getLinesInRange(*sources[sourceFileName], tik.TextRange{
			Start: start,
			End:   end,
		})
	}

	var canvas []renderLine

	for _, filename := range orderSource {
		source, _ := sources[filename]
		labels, _ := sourceLabels[filename]
		srcLines, _ := sourceLines[filename]

		annotations := make(map[int][]string)
		for _, label := range labels {
			tmp.Reset()
			// get the line info
			info := rangeToLineInfo(*source, label.Range)

			// pad the start of the line with spaces up to the beginning of the label range
			fmt.Fprint(&tmp, strings.Repeat(" ", info.Col))

			// add '^' across the label range
			fmt.Fprintf(&tmp, "└")
			for range label.Range.End - label.Range.Start - 2 {
				fmt.Fprint(&tmp, "─")
			}
			fmt.Fprintf(&tmp, "┴─")

			// add the note to the end
			fmt.Fprintf(&tmp, " %s", label.Note)
			annotations[info.Line] = append(annotations[info.Line], tmp.String())
		}

		maxLine := srcLines[len(srcLines)-1].Line
		r.gutterWidth = len(fmt.Sprintf("%d", maxLine)) + 1

		canvas = append(canvas, renderLine{LineNum: 0, Content: fmt.Sprintf("%s", filename)})

		for _, li := range srcLines {
			canvas = append(canvas, renderLine{LineNum: li.Line, Content: li.Content, IsSrc: true})
			if notes, ok := annotations[li.Line]; ok {
				for _, note := range notes {
					canvas = append(canvas, renderLine{LineNum: 0, Content: note, IsSrc: false})
				}
			}
		}

		fmt.Fprintf(&out, " ├─ %s:%d:%d\n", report.Location.FileName, report.Location.Line, report.Location.Col)
		fmt.Fprintf(&out, " │\n")
		if report.Message != "" {
			fmt.Fprintf(&out, " │%s %s\n", r.inGutter("•"), report.Message)
		}
		if len(report.Notes) > 0 {
			for _, note := range report.Notes {
				fmt.Fprintf(&out, " │%s %s\n", r.inGutter(""), note)
			}
			fmt.Fprintf(&out, " │\n")
		}
		for i, row := range canvas {
			if i == 0 {
				fmt.Fprintf(&out, " │%s ┌─ %s\n", r.inGutter(""), row.Content)
				fmt.Fprintf(&out, " │%s │\n", r.inGutter(""))
				continue
			}
			if row.IsSrc {
				fmt.Fprintf(&out, " │%s │ ", r.inGutter(fmt.Sprint(row.LineNum)))
				syntaxHighlight(&out, row, nil)
				fmt.Fprint(&out, "\n")
			} else {
				fmt.Fprintf(&out, " ┆%s ┆ %s\n", r.inGutter(""), row.Content)
			}
		}
	}

	return out.String()
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

	for content := range strings.SplitSeq(string(src.Raw), "\n") {
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
