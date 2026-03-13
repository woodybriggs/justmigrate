package parser

import (
	"fmt"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
	"woodybriggs/justmigrate/frontend/token"
)

func (p *SqliteParser) CreateStatement() ast.Statement {
	p.PushParseContext("create statement")
	defer p.PopParseContext()

	switch p.Peeked().Kind {
	case token.TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(false)
	case token.TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(false)
	case token.TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(false)
	case token.TokenKind_Keyword_INDEX:
		isUnique := false
		return p.CreateIndexStatement(isUnique)
	case token.TokenKind_Keyword_UNIQUE:
		isUnique := true
		return p.CreateIndexStatement(isUnique)
	case token.TokenKind_Keyword_VIRTUAL:
		return p.CreateVirtualTableStatement()
	case token.TokenKind_Keyword_TEMPORARY:
		return p.CreateTemporaryStatement()
	default:
		err := report.
			NewReport("parse error").
			WithLabels(report.Label{
				Source: p.Current().SourceCode,
				Range:  p.Current().SourceRange,
				Note:   "unknown token for create statement",
			},
			)
		p.ReportError(err)
		return nil
	}
}

func (p *SqliteParser) CreateTableStatement(isTemporary bool) *ast.CreateTable {
	p.PushParseContext("create table statement")
	defer p.PopParseContext()

	var temporaryKeyword *ast.Keyword = nil

	createKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CREATE))

	if isTemporary {
		temporaryKeyword = ast.MakeKeyword(p.Expect(token.TokenKind_Keyword_TEMPORARY))
	}

	tableKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_TABLE))

	ifnotexists := p.MaybeIfNotExists()

	tableIdent := p.CatalogObjectIdentifier()

	tableDefinition := p.TableDefinition()

	tableOptions := p.MaybeTableOptions()

	return ast.MakeCreateTable(
		createKeyword,
		temporaryKeyword,
		tableKeyword,
		ifnotexists,
		tableIdent,
		tableDefinition,
		tableOptions,
	)
}

func (p *SqliteParser) CreateViewStatement(false bool) ast.Statement {
	panic(fmt.Errorf("%w: CreateViewStatement", ErrNotImplemented))
}

func (p *SqliteParser) CreateTriggerStatement(false bool) ast.Statement {
	panic(fmt.Errorf("%w: CreateTriggerStatement", ErrNotImplemented))
}

func (p *SqliteParser) CreateIndexStatement(isUnique bool) ast.Statement {
	createKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CREATE))

	var uniqueKeyword *ast.Keyword = nil
	if p.Current().Kind == token.TokenKind_Keyword_UNIQUE {
		uniqueKeyword = ast.MakeKeyword(p.Current())
		p.Advance()
	}

	indexKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_INDEX))

	ifNotExists := p.MaybeIfNotExists()

	indexIdentifier := p.CatalogObjectIdentifier()

	onKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_ON))

	tableName := p.Identifier()

	lParen := p.Expect('(')

	indexedCols := []ast.IndexedColumn{}
	for !p.EndOfFile() {
		if p.Current().Kind == ')' {
			break
		} else if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else {
			indexedCol := p.IndexedColumn(true)
			indexedCols = append(indexedCols, indexedCol)
		}
	}

	rParen := p.Expect(')')

	var whereKeyword *ast.Keyword = nil
	var whereExpr ast.Expr = nil
	if p.Current().Kind == token.TokenKind_Keyword_WHERE {
		whereKeyword = ast.MakeKeyword(p.Current())
		p.Advance()
		whereExpr = p.Expr(0)
	}

	return ast.MakeCreateIndex(
		createKeyword,
		uniqueKeyword,
		indexKeyword,
		ifNotExists,
		indexIdentifier,
		onKeyword,
		tableName,
		lParen,
		indexedCols,
		rParen,
		whereKeyword,
		whereExpr,
	)
}

func (p *SqliteParser) CreateVirtualTableStatement() ast.Statement {
	panic(fmt.Errorf("%w: CreateVirtualTableStatement", ErrNotImplemented))
}

func (p *SqliteParser) CreateTemporaryStatement() ast.Statement {
	panic(fmt.Errorf("%w: CreateTemporaryStatement", ErrNotImplemented))
}

func (p *SqliteParser) MaybeIfNotExists() *ast.IfNotExists {
	if p.Current().Kind != token.TokenKind_Keyword_IF {
		return nil
	}
	ifKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_IF))
	notKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_NOT))
	existsKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_EXISTS))

	return ast.MakeIfNotExists(ifKeyword, notKeyword, existsKeyword)
}

func (p *SqliteParser) MaybeTableOptions() *ast.TableOptions {

	var withoutRowId *ast.WithoutRowId = nil
	var strict *ast.Keyword = nil

	for !p.EndOfFile() {
		if p.Current().Kind == token.TokenKind_Keyword_STRICT {
			strict = ast.MakeKeyword(p.Current())
			p.Advance()
		} else if p.Current().Kind == token.TokenKind_Keyword_WITHOUT {
			withoutRowId = p.WithoutRowId()
		}

		if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else {
			break
		}
	}

	return ast.MakeTableOptions(
		strict,
		withoutRowId,
	)
}

func (p *SqliteParser) WithoutRowId() *ast.WithoutRowId {
	withoutKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_WITHOUT))
	rowidKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_ROWID))
	return ast.MakeWithoutRowId(
		withoutKeyword,
		rowidKeyword,
	)
}
