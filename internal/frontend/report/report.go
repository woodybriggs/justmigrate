package report

import (
	"fmt"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/lexer"
	"justmigrate/internal/frontend/token"
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
	tr := token.NewTextRange()
	getSourceRangeFromExpr(expr, tr)

	return Label{
		Source: source,
		Range:  *tr,
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
	case *ast.LiteralSignedInteger:
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
	default:
		panic("getSourceRangeFromExpr: unhandled expr type")
	}
}

func LocationFromExpr(expr ast.Expr) token.Location {
	switch expr := expr.(type) {
	case *ast.BinaryOp:
		return expr.Operator.FileLoc
	case *ast.Identifier:
		return expr.FileLoc
	case *ast.LiteralNull:
		return expr.Token.FileLoc
	case *ast.LiteralBoolean:
		return expr.Token.FileLoc
	case *ast.LiteralFloat:
		return expr.Token.FileLoc
	case *ast.LiteralSignedInteger:
		return expr.Token.FileLoc
	case *ast.LiteralString:
		return expr.Token.FileLoc
	case *ast.LiteralUnsignedInteger:
		return expr.Token.FileLoc
	case *ast.FunctionCall:
		return expr.Name.FileLoc
	case *ast.CaseExpression:
		return LocationFromExpr(expr.Cases[0].When)
	case *ast.ColumnName:
		return expr.Column.FileLoc
	case ast.ExprList:
		return LocationFromExpr(expr[0])
	case *ast.ParseError:
		return expr.ConsumedTokens[0].FileLoc
	default:
		panic("LocationFromExpr: unhandled expr type")
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
