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

func LabelFromExpr(expr ast.Expr, note string) Label {

	source := getSourceFromExpr(expr)
	tr := token.TextRange{}
	getSourceRangeFromExpr(expr, &tr)

	return Label{
		Source: source,
		Range:  tr,
		Note:   note,
	}
}

func getSourceFromExpr(expr ast.Expr) lexer.SourceCode {
	switch expr := expr.(type) {
	case *ast.BinaryOp:
		return expr.Operator.SourceCode
	// case *ast.UnaryOp:
	// 	return expr.Operator.SourceCode
	case *ast.Identifier:
		return expr.SourceCode
	case *ast.LiteralBoolean:
		return expr.Token.SourceCode
	case *ast.LiteralFloat:
		return expr.Token.SourceCode
	case *ast.LiteralNull:
		return expr.Token.SourceCode
	case *ast.CaseExpression:
		return getSourceFromExpr(expr.Cases[0].When)
	default:
		panic("getSourceFromExpr: unhandled expr type")
	}
}

func getSourceRangeFromExpr(expr ast.Expr, tr *token.TextRange) {
	switch expr := expr.(type) {
	case *ast.BinaryOp:
		tr.ExtendBy(expr.Operator.SourceRange)
		getSourceRangeFromExpr(expr.Lhs, tr)
		getSourceRangeFromExpr(expr.Rhs, tr)
	case *ast.Identifier:
		tr.ExtendBy(expr.SourceRange)
	case *ast.LiteralNull:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.LiteralBoolean:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.LiteralFloat:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.LiteralSignedInteger:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.LiteralString:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.LiteralUnsignedInteger:
		tr.ExtendBy(expr.Token.SourceRange)
	case *ast.FunctionCall:
		tr.ExtendBy(expr.Name.SourceRange)
		for _, arg := range expr.Args {
			getSourceRangeFromExpr(arg, tr)
		}
	case *ast.CaseExpression:
		getSourceRangeFromExpr(expr.Operand, tr)
		for _, whenthen := range expr.Cases {
			getSourceRangeFromExpr(whenthen.When, tr)
			getSourceRangeFromExpr(whenthen.Then, tr)
		}
		getSourceRangeFromExpr(expr.Else, tr)
	}
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
