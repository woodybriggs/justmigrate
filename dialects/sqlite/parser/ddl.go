package sqlite

import (
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/core/tik"
)

func (p *SqliteParser) CreateStatement() ast.Statement {
	p.PushParseContext("create statement")
	defer p.PopParseContext()

	switch p.Peeked().Kind {
	case tik.TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(false)
	case tik.TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(false)
	case tik.TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(false)
	case tik.TokenKind_Keyword_INDEX:
		isUnique := false
		return p.CreateIndexStatement(isUnique)
	case tik.TokenKind_Keyword_UNIQUE:
		isUnique := true
		return p.CreateIndexStatement(isUnique)
	case tik.TokenKind_Keyword_VIRTUAL:
		return p.CreateVirtualTableStatement()
	case tik.TokenKind_Keyword_TEMPORARY:
		return p.CreateTemporaryStatement()
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.Current().SourceCode,
					Range:  p.Current().SourceRange,
					Note:   "unknown token for create statement",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *SqliteParser) CreateTableStatement(isTemporary bool) *ast.CreateTable {
	p.PushParseContext("create table statement")
	defer p.PopParseContext()

	var temporaryKeyword *ast.Keyword = nil

	createKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_CREATE))

	if isTemporary {
		temporaryKeyword = ast.MakeKeyword(p.Expect(tik.TokenKind_Keyword_TEMPORARY))
	}

	tableKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_TABLE))

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
	panic("unimplemented")
}

func (p *SqliteParser) CreateTriggerStatement(false bool) ast.Statement {
	panic("unimplemented")
}

func (p *SqliteParser) CreateIndexStatement(isUnique bool) ast.Statement {
	panic("unimplemented")
}

func (p *SqliteParser) CreateVirtualTableStatement() ast.Statement {
	panic("unimplemented")
}

func (p *SqliteParser) CreateTemporaryStatement() ast.Statement {
	panic("unimplemented")
}

func (p *SqliteParser) MaybeIfNotExists() *ast.IfNotExists {
	if p.Current().Kind != tik.TokenKind_Keyword_IF {
		return nil
	}
	ifKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_IF))
	notKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_NOT))
	existsKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_EXISTS))

	return ast.MakeIfNotExists(ifKeyword, notKeyword, existsKeyword)
}

func (p *SqliteParser) MaybeTableOptions() *ast.TableOptions {

	var withoutRowId *ast.WithoutRowId = nil
	var strict *ast.Keyword = nil

	for !p.EndOfFile() {
		if p.Current().Kind == tik.TokenKind_Keyword_STRICT {
			strict = ast.MakeKeyword(p.Current())
			p.Advance()
		} else if p.Current().Kind == tik.TokenKind_Keyword_WITHOUT {
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
	withoutKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_WITHOUT))
	rowidKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_ROWID))
	return ast.MakeWithoutRowId(
		withoutKeyword,
		rowidKeyword,
	)
}
