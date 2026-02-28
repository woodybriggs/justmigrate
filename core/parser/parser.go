package parser

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/core/tik"
	"woodybriggs/justmigrate/datastructures"
)

type ParseContext struct {
	Name          string
	StartingToken tik.Token
	EndingToken   tik.Token
}

type Parser struct {
	lexer        *luther.Lexer
	currentToken tik.Token
	peekedToken  tik.Token

	errors   map[tik.TextRange]report.Report
	warnings map[tik.TextRange]report.Report

	parseContext datastructures.Stack[ParseContext]
}

func NewParser(lexer *luther.Lexer) *Parser {

	result := &Parser{
		lexer:        lexer,
		errors:       map[tik.TextRange]report.Report{},
		warnings:     map[tik.TextRange]report.Report{},
		parseContext: datastructures.Stack[ParseContext]{},
	}

	// prime the parser
	result.currentToken = lexer.NextToken()
	result.peekedToken = lexer.PeekToken()

	return result
}

func (p *Parser) Errors() []report.Report {
	return slices.Collect(maps.Values(p.errors))
}

func (p *Parser) Advance() {
	p.currentToken = p.lexer.NextToken()
	p.peekedToken = p.lexer.PeekToken()
}

func (p *Parser) Expect(kind tik.TokenKind) tik.Token {
	if p.currentToken.Kind == kind {
		token := p.currentToken
		p.Advance()
		return token
	}

	if err, ok := p.errors[p.currentToken.SourceRange]; ok {
		panic(err)
	}

	parseContext, ok := p.parseContext.Top()

	if ok {
		p.ReportError(
			report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.currentToken.SourceCode,
						Range:  p.currentToken.SourceRange,
						Note:   fmt.Sprintf("expected '%s' got '%s'", kind.DebugString(), p.currentToken.DebugString()),
					},
				}).
				WithNotes(
					[]string{fmt.Sprintf("attempting to parse %s", parseContext.Name)},
				),
		)
	} else {
		p.ReportError(
			report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.currentToken.SourceCode,
						Range:  p.currentToken.SourceRange,
						Note:   fmt.Sprintf("expected '%s' got '%s'", kind.DebugString(), p.currentToken.DebugString()),
					},
				}),
		)
	}

	if p.peekedToken.Kind == kind {
		builder := strings.Builder{}
		builder.WriteString(p.currentToken.String())
		p.Advance()
		builder.WriteString(p.currentToken.LeadingTrivia)
		p.currentToken.LeadingTrivia = builder.String()
		realToken := p.currentToken
		p.Advance() // Consume '('
		return realToken
	}

	costSynthesis := tik.Token{Kind: kind}.InsertionCost()
	costDeletion := p.currentToken.DeletionCost()

	if costSynthesis <= costDeletion {
		// p.Advance()
		return tik.Token{
			Kind: kind,
		}
	}

	builder := strings.Builder{}
	builder.WriteString(p.currentToken.String())
	p.Advance()
	builder.WriteString(p.currentToken.LeadingTrivia)
	p.currentToken.LeadingTrivia = builder.String()
	return p.currentToken
}

func (p *Parser) Current() tik.Token {
	return p.currentToken
}

func (p *Parser) Peeked() tik.Token {
	return p.peekedToken
}

func (p *Parser) ReportError(report *report.Report) {

	if _, has := p.errors[p.currentToken.SourceRange]; has {
		panic(report)
	}

	p.errors[p.currentToken.SourceRange] = *report
}

func (p *Parser) ReportWarning(report *report.Report) {
	p.warnings[p.currentToken.SourceRange] = *report
}

func (p *Parser) Synchronize(syncTokens []tik.TokenKind) {
	for !p.lexer.Eof() {
		if v := slices.Index(syncTokens, p.currentToken.Kind); v > -1 {
			p.Advance()
			return
		} else {
			p.Advance()
		}
	}
}

func (p *Parser) PushParseContext(name string) {
	p.parseContext.Push(
		ParseContext{
			Name:          name,
			StartingToken: p.currentToken,
			EndingToken:   p.currentToken,
		},
	)
}

func (p *Parser) PopParseContext() {
	if len(p.parseContext.Data) > 0 {
		p.parseContext.Data[len(p.parseContext.Data)-1].EndingToken = p.currentToken
	}
	p.parseContext.Pop()
}

func (p *Parser) EndOfFile() bool {
	return p.Current().Kind == tik.TokenKind_EOF
}

func (p *Parser) CatalogObjectIdentifier() *ast.CatalogObjectIdentifier {
	p.PushParseContext("catalog object identifier")
	defer p.PopParseContext()

	schemaOrTable := p.Identifier()

	if p.Current().Kind != tik.TokenKind_Period {
		tableName := schemaOrTable
		return ast.MakeCatalogObjectIdentifier(nil, tableName)
	}
	p.Advance()

	schemaName := schemaOrTable
	tableName := p.Identifier()

	return ast.MakeCatalogObjectIdentifier(
		&schemaName,
		tableName,
	)
}

func (p *Parser) Identifier() ast.Identifier {
	p.PushParseContext("identifier")
	defer p.PopParseContext()
	return ast.Identifier(p.Expect(tik.TokenKind_Identifier))
}

func (p *Parser) MaybeCollation() *ast.Collation {

	if p.Current().Kind != tik.TokenKind_Keyword_COLLATE {
		return nil
	}
	collateKeyword := ast.Keyword(p.Current())
	name := p.Identifier()

	return ast.MakeCollation(
		collateKeyword,
		name,
	)
}

type PrattParser interface {
	Term() ast.Expr
	OperatorBindingPower(token tik.Token) (bp ast.BindingPower, found bool)
}

func (p *Parser) Expr(
	minBindingPower int,
	prattParser PrattParser,
) ast.Expr {
	lhs := prattParser.Term()

	for !p.EndOfFile() {
		bp, found := prattParser.OperatorBindingPower(p.Current())
		if !found {
			return lhs
		}
		if bp.L < minBindingPower {
			break
		}

		op := p.Current()
		p.Advance()

		rhs := p.Expr(bp.R, prattParser)

		binaryOp := ast.MakeBinaryOpExpr(
			lhs,
			op,
			rhs,
		)

		lhs = binaryOp
	}

	return lhs
}
