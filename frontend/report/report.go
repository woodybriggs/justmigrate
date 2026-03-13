package report

import (
	"fmt"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/lexer"
	"woodybriggs/justmigrate/frontend/token"
)

type Label struct {
	Source lexer.SourceCode
	Range  token.TextRange
	Note   string
}

func LabelFromToken(token token.Token, note string) Label {
	return Label{
		Source: token.SourceCode,
		Range:  token.SourceRange,
		Note:   note,
	}
}

func LabelFromIdentifier(ident ast.Identifier, note string) Label {
	return LabelFromToken(token.Token(ident), note)
}

func LabelFromKeyword(keyword ast.Keyword, note string) Label {
	return LabelFromToken(token.Token(keyword), note)
}

func (label Label) String() string {
	return fmt.Sprintf("%s:%d:%d %s", label.Source.FileName, label.Range.Start, label.Range.End, label.Note)
}

type Report struct {
	Kind     string
	Location token.Location
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

func (report *Report) WithLocation(location token.Location) *Report {
	report.Location = location
	return report
}

func (report *Report) WithMessage(message string) *Report {
	report.Message = message
	return report
}

func (report *Report) WithLabels(labels ...Label) *Report {
	report.Labels = append(report.Labels, labels...)
	return report
}

func (report *Report) WithNotes(notes ...string) *Report {
	report.Notes = append(report.Notes, notes...)
	return report
}
