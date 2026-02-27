package sqlite

import (
	"errors"
	"fmt"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/parser"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/core/tik"
)

var ErrNotImplemented = errors.New("not implemented")

type SqliteParser struct {
	*parser.Parser
}

func NewSqliteParser(lexer *luther.Lexer) *SqliteParser {
	core := parser.NewParser(lexer)
	return &SqliteParser{
		Parser: core,
	}
}

func (p *SqliteParser) TableDefinition() *ast.TableDefinition {
	p.PushParseContext("table definition")
	defer p.PopParseContext()

	lParen := p.Expect('(')

	columnDefs := p.ColumnDefinitions()
	tableConstraints := p.TableConstraints()

	rParen := p.Expect(')')

	return ast.MakeTableDefinition(
		lParen,
		columnDefs,
		tableConstraints,
		rParen,
	)
}

func (p *SqliteParser) TableConstraints() []ast.TableConstraint {
	p.PushParseContext("table constraints")
	defer p.PopParseContext()

	result := []ast.TableConstraint{}

	for !p.EndOfFile() {
		if p.Current().Kind == '(' {
			break
		} else if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else {
			tableConstraint := p.TableConstraint()
			result = append(result, tableConstraint)
		}
	}

	return result
}

func (p *SqliteParser) TableConstraint() ast.TableConstraint {

	p.PushParseContext("table constraint")
	defer p.PopParseContext()

	constraintName := p.MaybeConstraintName()
	if constraintName == nil {
		p.ReportWarning(
			report.NewReport("warning").
				WithMessage("unnamed table constraint").
				WithLabels([]report.Label{
					{
						Source: p.Current().SourceCode,
						Range:  p.Current().SourceRange,
						Note:   "CONSTRAINT constraint_name",
					},
				}).
				WithNotes([]string{
					"by adding a constraint name, we can detect changes of table constraints, and migrate them appropriately.",
				}),
		)
	}

	switch p.Current().Kind {
	case tik.TokenKind_Keyword_PRIMARY:
		return p.TableConstraint_PrimaryKey(constraintName)
	case tik.TokenKind_Keyword_FOREIGN:
		return p.TableConstraint_ForeignKey(constraintName)
	case tik.TokenKind_Keyword_CHECK:
		return p.TableConstraint_Check(constraintName)
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.Current().SourceCode,
					Range:  p.Current().SourceRange,
					Note:   "unexpected token for table constraint",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *SqliteParser) TableConstraint_PrimaryKey(constraintName *ast.ConstraintName) ast.TableConstraint {
	p.PushParseContext("primary key table constraint")
	defer p.PopParseContext()

	var autoincrement *ast.Keyword = nil

	primaryKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_PRIMARY))
	keyKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_KEY))
	lParen := p.Expect('(')

	indexedCols := []ast.IndexedColumn{}

	// take the first one manually incase it is followed by autoincrement
	indexedCol := p.IndexedColumn(false)
	indexedCols = append(indexedCols, indexedCol)

	if p.Current().Kind == tik.TokenKind_Keyword_AUTOINCREMENT {
		autoincrement = ast.MakeKeyword(p.Current())
		p.Advance()
	}

	if autoincrement == nil {
		for !p.EndOfFile() {
			if p.Current().Kind == '(' {
				break
			} else if p.Current().Kind == ',' {
				p.Advance()
				continue
			} else {
				indexedCol := p.IndexedColumn(false)
				indexedCols = append(indexedCols, indexedCol)
			}
		}
	}

	rParen := p.Expect(')')

	conflictClause := p.MaybeConflictClause()

	return ast.MakeTableConstraintPrimaryKey(
		constraintName,
		primaryKeyword,
		keyKeyword,
		lParen,
		indexedCols,
		rParen,
		conflictClause,
		autoincrement,
	)
}

func (p *SqliteParser) IndexedColumn(allowExpressions bool) ast.IndexedColumn {

	p.PushParseContext("indexed column")
	defer p.PopParseContext()

	var expr ast.Expr = nil
	if allowExpressions {
		expr = p.Expr(0)
	} else {
		tmp := p.Identifier()
		expr = &tmp
	}

	collation := p.MaybeCollation()

	order := p.MaybeOrderKeyword()

	return ast.IndexedColumn{
		Subject:   expr,
		Collation: collation,
		Order:     order,
	}
}

func (p *SqliteParser) TableConstraint_ForeignKey(constraintName *ast.ConstraintName) ast.TableConstraint {
	p.PushParseContext("foreign key table constraint")
	defer p.PopParseContext()

	foreign := ast.Keyword(p.Expect(tik.TokenKind_Keyword_FOREIGN))
	key := ast.Keyword(p.Expect(tik.TokenKind_Keyword_KEY))

	lParen := p.Expect('(')

	columnNames := []ast.Identifier{}
	for !p.EndOfFile() {
		if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else if p.Current().Kind == ')' {
			break
		} else {
			columnName := p.Identifier()
			columnNames = append(columnNames, columnName)
		}
	}

	rParen := p.Expect(')')

	fkClause := p.ForeignKeyClause()
	return ast.MakeTableConstraintForeignKey(
		constraintName,
		foreign,
		key,
		lParen,
		columnNames,
		rParen,
		fkClause,
	)
}

func (p *SqliteParser) ForeignKeyClause() *ast.ForeignKeyClause {

	referencesKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_REFERENCES))

	foreignTable := p.CatalogObjectIdentifier()

	lParen := p.Expect('(')

	columns := []ast.Identifier{}

	for !p.EndOfFile() {
		if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else if p.Current().Kind == ')' {
			break
		} else {
			column := p.Identifier()
			columns = append(columns, column)
		}
	}

	rParen := p.Expect(')')

	var deferrable *ast.ForeignKeyDeferrable = nil
	var matchName *ast.Identifier = nil
	actions := []ast.ForeignKeyActionTrigger{}

	for !p.EndOfFile() {
		if p.Current().Kind == tik.TokenKind_Keyword_ON {
			action := p.ForeignKeyAction()
			actions = append(actions, action)
		} else if p.Current().Kind == tik.TokenKind_Keyword_MATCH {
			ident := p.Identifier()
			matchName = &ident
		} else if p.Current().Kind == tik.TokenKind_Keyword_NOT {
			deferrable = p.ForeignKeyDeferrable()
		} else if p.Current().Kind == tik.TokenKind_Keyword_DEFERRABLE {
			deferrable = p.ForeignKeyDeferrable()
		} else {
			break
		}
	}

	return ast.MakeForeignKeyClause(
		referencesKeyword,
		*foreignTable,
		lParen,
		columns,
		rParen,
		actions,
		matchName,
		deferrable,
	)
}

func (p *SqliteParser) TableOptions() *ast.TableOptions {
	panic("unimplemented")
}

func (p *SqliteParser) ColumnDefinitions() []ast.ColumnDefinition {
	p.PushParseContext("column definitions")
	defer p.PopParseContext()

	definitions := []ast.ColumnDefinition{}

	for !p.EndOfFile() {
		if p.Current().Kind == ',' {
			p.Advance()
			continue
		} else if p.Current().Kind == ')' {
			break
		}

		columnDef := p.ColumnDefinition()
		definitions = append(definitions, *columnDef)
	}

	return definitions
}

func (p *SqliteParser) ColumnDefinition() *ast.ColumnDefinition {
	p.PushParseContext("column definition")
	defer p.PopParseContext()

	columnName := p.Identifier()
	typeName := p.Identifier()
	columnConstraints := p.ColumnConstraints()

	return ast.MakeColumnDefinition(
		columnName,
		typeName,
		columnConstraints,
	)
}

func (p *SqliteParser) ColumnConstraints() []ast.ColumnConstraint {

	p.PushParseContext("column constraints")
	defer p.PopParseContext()

	result := []ast.ColumnConstraint{}

	for p.Current().Kind != ',' && p.Current().Kind != ')' && !p.EndOfFile() {
		columnConstraint := p.ColumnConstraint()
		result = append(result, columnConstraint)
	}

	return result
}

func (p *SqliteParser) ColumnConstraint() ast.ColumnConstraint {
	p.PushParseContext("column constraint")
	defer p.PopParseContext()

	constraintName := p.MaybeConstraintName()

	switch p.Current().Kind {
	case tik.TokenKind_Keyword_PRIMARY:
		return p.ColumnConstraint_PrimaryKey(constraintName)
	// case tik.TokenKind_Keyword_NOT:
	// 	return p.ColumnConstraint_NotNull(constraintName)
	// case tik.TokenKind_Keyword_DEFAULT:
	// 	return p.ColumnConstraint_Default(constraintName)
	// case tik.TokenKind_Keyword_UNIQUE:
	// 	return p.ColumnConstraint_Unique(constraintName)
	// case tik.TokenKind_Keyword_COLLATE:
	// 	return p.ColumnConstraint_Collate(constraintName)
	// case tik.TokenKind_Keyword_CHECK:
	// 	return p.ColumnConstraint_Check(constraintName)
	// case tik.TokenKind_Keyword_AS:
	// 	return p.ColumnConstraint_Generated(constraintName)
	// case tik.TokenKind_Keyword_GENERATED:
	// 	return p.ColumnConstraint_Generated(constraintName)
	default:
		{
			p.ReportError(
				report.
					NewReport("parse error").
					WithLabels([]report.Label{
						{
							Source: p.Current().SourceCode,
							Range:  p.Current().SourceRange,
							Note:   "unexpected token at start of column constraint",
						},
					}),
			)
			return nil
		}
	}
}

func (p *SqliteParser) ColumnConstraint_PrimaryKey(constraintName *ast.ConstraintName) *ast.ColumnConstraint_PrimaryKey {
	p.PushParseContext("primary key column constraint")
	defer p.PopParseContext()

	primaryKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_PRIMARY))
	keyKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_KEY))

	order := p.MaybeOrderKeyword()
	conflictClause := p.MaybeConflictClause()

	var autoincrement *ast.Keyword = nil
	if p.Current().Kind == tik.TokenKind_Keyword_AUTOINCREMENT {
		autoincrement = ast.MakeKeyword(p.Current())
		p.Advance()
	}

	return ast.MakeColumnConstraintPrimaryKey(
		constraintName,
		primaryKeyword,
		keyKeyword,
		order,
		conflictClause,
		autoincrement,
	)
}

func (p *SqliteParser) MaybeConstraintName() *ast.ConstraintName {
	p.PushParseContext("constraint name")
	defer p.PopParseContext()

	// we may or may not have CONSTRAINT keyword here so have a look-y and check-y
	if p.Current().Kind != tik.TokenKind_Keyword_CONSTRAINT {
		return nil
	}
	constraintKeyword := ast.Keyword(p.Current())
	p.Advance()

	name := p.Identifier()

	return &ast.ConstraintName{
		ConstraintKeyword: constraintKeyword,
		Name:              name,
	}
}

func (p *SqliteParser) MaybeConflictClause() *ast.ConflictClause {
	if p.Current().Kind != tik.TokenKind_Keyword_ON {
		return nil
	}
	onKeyword := ast.Keyword(p.Current())
	p.Advance()

	conflictKeyword := ast.Keyword(p.Expect(tik.TokenKind_Keyword_CONFLICT))

	switch p.Current().Kind {
	case tik.TokenKind_Keyword_ROLLBACK,
		tik.TokenKind_Keyword_ABORT,
		tik.TokenKind_Keyword_FAIL,
		tik.TokenKind_Keyword_IGNORE,
		tik.TokenKind_Keyword_REPLACE:
		{
			actionKeyword := ast.Keyword(p.Current())
			return ast.MakeConflictClause(
				onKeyword,
				conflictKeyword,
				actionKeyword,
			)
		}
	default:
		{
			p.ReportError(
				report.NewReport("parse error").
					WithLabels([]report.Label{
						{
							Source: p.Current().SourceCode,
							Range:  p.Current().SourceRange,
							Note:   "expected 'rollback', 'abort', 'fail', 'ignore' or 'replace'",
						},
					}).
					WithMessage(fmt.Sprintf("got '%s'", p.Current().Text)),
			)
			return nil
		}
	}
}

func (p *SqliteParser) MaybeOrderKeyword() *ast.Keyword {
	switch p.Current().Kind {
	case tik.TokenKind_Keyword_ASC:
		fallthrough
	case tik.TokenKind_Keyword_DESC:
		return ast.MakeKeyword(p.Current())
	default:
		return nil
	}
}
