package report

import (
	"fmt"
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
	Kind     string
	Code     int
	Location tik.Location
	Message  string
	Labels   []Label
	Notes    []string
}

func (r *Report) Error() string {
	renderer := Renderer{}
	return renderer.Render(*r)
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

func (report *Report) WithLocation(location tik.Location) *Report {
	report.Location = location
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
