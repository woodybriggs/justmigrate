package parser

import (
	"errors"
	"fmt"
	"strconv"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/lexer"
	"woodybriggs/justmigrate/frontend/parser"
	"woodybriggs/justmigrate/frontend/report"
	"woodybriggs/justmigrate/frontend/token"
)

var ErrNotImplemented = errors.New("not implemented")

type SqliteParser struct {
	*parser.Parser
}

func NewSqliteParser(lexer *lexer.Lexer) *SqliteParser {
	return &SqliteParser{
		Parser: parser.NewParser(lexer),
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
		if p.Current().Kind == ')' {
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
				WithLocation(p.Current().FileLoc).
				WithMessage("unnamed table constraint").
				WithLabels(
					report.LabelFromToken(p.Current(), "CONSTRAINT constraint name"),
				).
				WithNotes(
					"by adding a constraint name, we can detect changes of table" +
						"constraints, and migrate them appropriately.",
				),
		)
	}

	switch p.Current().Kind {
	case token.TokenKind_Keyword_PRIMARY:
		return p.TableConstraint_PrimaryKey(constraintName)
	case token.TokenKind_Keyword_FOREIGN:
		return p.TableConstraint_ForeignKey(constraintName)
	case token.TokenKind_Keyword_CHECK:
		return p.TableConstraint_Check(constraintName)
	default:
		err := report.
			NewReport("parse error").
			WithLocation(p.Current().FileLoc).
			WithLabels(report.LabelFromToken(p.Current(), "unexpected token for table constraint"))
		p.ReportError(err)
		return nil
	}
}

func (p *SqliteParser) TableConstraint_PrimaryKey(constraintName *ast.ConstraintName) ast.TableConstraint {
	p.PushParseContext("primary key table constraint")
	defer p.PopParseContext()

	var autoincrement *ast.Keyword = nil

	primaryKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_PRIMARY))
	keyKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_KEY))
	lParen := p.Expect('(')

	indexedCols := []ast.IndexedColumn{}

	// take the first one manually incase it is followed by autoincrement
	indexedCol := p.IndexedColumn(false)
	indexedCols = append(indexedCols, indexedCol)

	if p.Current().Kind == token.TokenKind_Keyword_AUTOINCREMENT {
		autoincrement = ast.MakeKeyword(p.Current())
		p.Advance()
	}

	if autoincrement == nil {
		for !p.EndOfFile() {
			if p.Current().Kind == ')' {
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

	foreignKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_FOREIGN))
	keyKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_KEY))

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
		foreignKeyword,
		keyKeyword,
		lParen,
		columnNames,
		rParen,
		fkClause,
	)
}

func (p *SqliteParser) ForeignKeyClause() *ast.ForeignKeyClause {

	referencesKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_REFERENCES))

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
	actions := []ast.ForeignKeyAction{}

	for !p.EndOfFile() {
		if p.Current().Kind == token.TokenKind_Keyword_ON {
			action := p.ForeignKeyAction()
			actions = append(actions, action)
		} else if p.Current().Kind == token.TokenKind_Keyword_MATCH {
			ident := p.Identifier()
			matchName = &ident
		} else if p.Current().Kind == token.TokenKind_Keyword_NOT {
			deferrable = p.ForeignKeyDeferrable()
		} else if p.Current().Kind == token.TokenKind_Keyword_DEFERRABLE {
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

func (p *SqliteParser) ForeignKeyAction() ast.ForeignKeyAction {
	onKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_ON))

	switch p.Current().Kind {
	case token.TokenKind_Keyword_UPDATE:
		updateKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_UPDATE))
		do := p.ForeignKeyActionDo()
		return ast.MakeForeignKeyUpdateAction(
			onKeyword,
			updateKeyword,
			do,
		)
	case token.TokenKind_Keyword_DELETE:
		deleteKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_DELETE))
		do := p.ForeignKeyActionDo()
		return ast.MakeForeignKeyDeleteAction(
			onKeyword,
			deleteKeyword,
			do,
		)
	default:
		p.ReportError(
			report.NewReport("parse error").
				WithLocation(p.Current().FileLoc).
				WithLabels(report.LabelFromToken(p.Current(), "here")).
				WithNotes("unexpected token when parsing foreign key action, expected 'update' or 'delete'"),
		)
		return nil
	}
}

func (p *SqliteParser) ForeignKeyActionDo() ast.ForeignKeyActionDo {

	switch p.Current().Kind {
	case token.TokenKind_Keyword_SET:
		setKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_SET))
		switch p.Current().Kind {
		case token.TokenKind_Keyword_NULL:
			nullKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_NULL))
			return ast.MakeForeignKeyActionSetNull(setKeyword, nullKeyword)
		case token.TokenKind_Keyword_DEFAULT:
			defaultKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_DEFAULT))
			return ast.MakeForeignKeyActionSetDefault(setKeyword, defaultKeyword)
		default:
			p.ReportError(
				report.NewReport("parse error").
					WithLocation(p.Current().FileLoc).
					WithLabels(report.LabelFromToken(p.Current(), "here")).
					WithMessage("expected 'null' or 'default' for set action of foreign key action"),
			)
			return nil
		}
	case token.TokenKind_Keyword_NO:
		noKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_NO))
		actionKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_ACTION))
		return ast.MakeForeignKeyActionNoAction(noKeyword, actionKeyword)
	case token.TokenKind_Keyword_RESTRICT:
		restrictKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_RESTRICT))
		return ast.MakeForeignKeyActionRestrict(restrictKeyword)
	case token.TokenKind_Keyword_CASCADE:
		cascadeKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CASCADE))
		return ast.MakeForeignKeyActionCascade(cascadeKeyword)
	default:
		p.ReportError(
			report.NewReport("parse error").
				WithLocation(p.Current().FileLoc).
				WithLabels(report.LabelFromToken(p.Current(), "here")).
				WithMessage("expected action: one of ('set null', 'set default', 'no action', 'restrict', 'cascade') for foreign key action"),
		)
		return nil
	}
}

func (p *SqliteParser) ForeignKeyDeferrable() *ast.ForeignKeyDeferrable {
	var notKeyword *ast.Keyword = nil
	if p.Current().Kind == token.TokenKind_Keyword_NOT {
		notKeyword = ast.MakeKeyword(p.Current())
		p.Advance()
	}

	deferrableKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_DEFERRABLE))

	var initiallyKeyword *ast.Keyword = nil
	var initiallyValue *ast.Keyword = nil
	if p.Current().Kind == token.TokenKind_Keyword_INITIALLY {
		initiallyKeyword = ast.MakeKeyword(p.Current())
		p.Advance()

		switch p.Current().Kind {
		case token.TokenKind_Keyword_DEFERRED:
			initiallyValue = ast.MakeKeyword(p.Current())
			p.Advance()
		case token.TokenKind_Keyword_IMMEDIATE:
			initiallyValue = ast.MakeKeyword(p.Current())
			p.Advance()
		default:
			p.ReportError(
				report.
					NewReport("parse errors").
					WithLocation(p.Current().FileLoc).
					WithLabels(report.LabelFromToken(p.Current(), "here")).
					WithMessage("expected value for 'deferrable initially' one of ('deferred' or 'immediate')"),
			)
			return nil
		}
	}

	return ast.MakeForeignKeyDeferrable(
		notKeyword,
		deferrableKeyword,
		initiallyKeyword,
		initiallyValue,
	)
}

func (p *SqliteParser) TableConstraint_Check(constraintName *ast.ConstraintName) ast.TableConstraint {
	checkKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CHECK))

	lParen := p.Expect('(')

	expr := p.Expr(0)

	rParen := p.Expect(')')

	return ast.MakeTableConstraintCheck(
		constraintName,
		checkKeyword,
		lParen,
		expr,
		rParen,
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
		if p.Current().Kind == ')' {
			break
		} else if isTableConstraintStartingToken(p.Current()) {
			break
		} else if p.Current().Kind == ',' {
			p.Advance()
		} else {
			columnDef := p.ColumnDefinition()
			definitions = append(definitions, *columnDef)
		}
	}

	return definitions
}

func (p *SqliteParser) ColumnDefinition() *ast.ColumnDefinition {
	p.PushParseContext("column definition")
	defer p.PopParseContext()

	columnName := p.Identifier()
	typeName := p.MaybeTypeName()
	columnConstraints := p.ColumnConstraints()

	return ast.MakeColumnDefinition(
		columnName,
		typeName,
		columnConstraints,
	)
}

func (p *SqliteParser) MaybeTypeName() *ast.TypeName {
	if p.Current().Kind != token.TokenKind_Identifier {
		return nil
	}

	name := p.Identifier()
	var arg0 ast.NumericLiteral = nil
	var arg1 ast.NumericLiteral = nil

	if p.Current().Kind == '(' {
		p.Advance()
		arg0 = p.SignedNumber()

		if p.Current().Kind == ',' {
			p.Advance()
			arg1 = p.SignedNumber()
		}

		p.Expect(')')
	}

	return ast.MakeTypeName(name, arg0, arg1)
}

func (p *SqliteParser) SignedNumber() ast.NumericLiteral {

	if p.Current().Kind == '+' {
		p.Advance()
	}

	var negate *token.Token = nil
	if p.Current().Kind == '-' {
		tok := p.Current()
		negate = &tok
		p.Advance()
	}

	return p.LiteralNumericLiteral(negate)
}

func (p *SqliteParser) LiteralNumericLiteral(negate *token.Token) ast.NumericLiteral {
	switch tok := p.Current(); tok.Kind {
	case token.TokenKind_IntegerNumericLiteral:
		p.Advance()
		val, err := strconv.ParseInt(tok.Text, 10, 64)
		if err != nil {
			p.Parser.ReportError(
				report.NewReport("parse error").
					WithLocation(p.Current().FileLoc).
					WithLabels(report.LabelFromToken(p.Current(), "here")).
					WithNotes(err.Error()),
			)
			return ast.MakeLiteralSignedInteger(tok, 0)
		}
		if negate != nil {
			val = val * -1
		}
		return ast.MakeLiteralSignedInteger(tok, val)
	case token.TokenKind_FloatNumericLiteral:
		p.Advance()
		val, err := strconv.ParseFloat(tok.Text, 64)
		if err != nil {
			p.Parser.ReportError(report.NewReport("parse error").
				WithLocation(p.Current().FileLoc).
				WithNotes(err.Error()).
				WithLabels(report.LabelFromToken(p.Current(), "here")),
			)
			return ast.MakeLiteralFloat(tok, 0)
		}
		if negate != nil {
			val = val * -1
		}
		return ast.MakeLiteralFloat(tok, val)
	default:
		p.ReportError(report.NewReport("parse error").
			WithLocation(p.Current().FileLoc).
			WithNotes("expected numeric literal (signed integer or float)").
			WithLabels(report.LabelFromToken(p.Current(), "here")),
		)

		// if the next token is a numeric literal, then we can skip the token and try parse again
		if p.Peeked().Kind == token.TokenKind_IntegerNumericLiteral || p.Peeked().Kind == token.TokenKind_FloatNumericLiteral {
			p.Advance()
			return p.LiteralNumericLiteral(negate)
		}

		// otherwise synthesize the node with 0 value
		return ast.MakeLiteralSignedInteger(tok, 0)
	}
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

func isTableConstraintStartingToken(tok token.Token) bool {
	switch tok.Kind {
	case token.TokenKind_Keyword_CONSTRAINT:
		return true
	case token.TokenKind_Keyword_PRIMARY:
		return true
	case token.TokenKind_Keyword_UNIQUE:
		return true
	case token.TokenKind_Keyword_CHECK:
		return true
	case token.TokenKind_Keyword_FOREIGN:
		return true
	default:
		return false
	}
}

func isColumnConstraintStartingToken(tok token.Token) bool {
	switch tok.Kind {
	case token.TokenKind_Keyword_CONSTRAINT:
		return true
	case token.TokenKind_Keyword_PRIMARY:
		return true
	case token.TokenKind_Keyword_NOT:
		return true
	case token.TokenKind_Keyword_DEFAULT:
		return true
	case token.TokenKind_Keyword_UNIQUE:
		return true
	case token.TokenKind_Keyword_COLLATE:
		return true
	case token.TokenKind_Keyword_CHECK:
		return true
	case token.TokenKind_Keyword_AS:
		return true
	case token.TokenKind_Keyword_GENERATED:
		return true
	default:
		return false
	}
}

func (p *SqliteParser) ColumnConstraint() ast.ColumnConstraint {
	p.PushParseContext("column constraint")
	defer p.PopParseContext()

	constraintName := p.MaybeConstraintName()

	switch p.Current().Kind {
	case token.TokenKind_Keyword_PRIMARY:
		return p.ColumnConstraint_PrimaryKey(constraintName)
	case token.TokenKind_Keyword_REFERENCES:
		return p.ColumnConstraint_ForeignKey(constraintName)
	case token.TokenKind_Keyword_NOT:
		return p.ColumnConstraint_NotNull(constraintName)
	case token.TokenKind_Keyword_DEFAULT:
		return p.ColumnConstraint_Default(constraintName)
		//	case token.TokenKind_Keyword_UNIQUE:
		//		return p.ColumnConstraint_Unique(constraintName)
		//	case token.TokenKind_Keyword_COLLATE:
		//		return p.ColumnConstraint_Collate(constraintName)
	case token.TokenKind_Keyword_CHECK:
		return p.ColumnConstraint_Check(constraintName)
		//	case token.TokenKind_Keyword_AS:
		//		return p.ColumnConstraint_Generated(constraintName)
		//	case token.TokenKind_Keyword_GENERATED:
		//		return p.ColumnConstraint_Generated(constraintName)
	default:
		{
			err :=
				report.
					NewReport("parse error").
					WithLocation(p.Current().FileLoc).
					WithLabels(report.LabelFromToken(p.Current(), "here")).
					WithNotes("unexpected token at start of column constraint")

			// if the next token is a valid constraint name, then we can skip the current and try parse again.
			if isColumnConstraintStartingToken(p.Peeked()) {
				err.WithNotes("faliure token will be skipped")
				p.Advance()
				return p.ColumnConstraint()
			}

			// otherwise we need to pretend that we got something, check NULL constraint seems to be safest.
			return ast.MakeColumnConstraintCheck(
				constraintName,
				ast.Keyword{Text: "CHECK"},
				ast.MakeLiteralNull(token.Token{Text: "NULL"}),
			)
		}
	}
}

func (p *SqliteParser) ColumnConstraint_ForeignKey(constraintName *ast.ConstraintName) *ast.ColumnConstraint_ForeignKey {
	clause := p.ForeignKeyClause()

	return ast.MakeColumnConstraintForeignKey(
		constraintName,
		*clause,
	)
}

func (p *SqliteParser) ColumnConstraint_PrimaryKey(constraintName *ast.ConstraintName) *ast.ColumnConstraint_PrimaryKey {
	p.PushParseContext("primary key column constraint")
	defer p.PopParseContext()

	primaryKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_PRIMARY))
	keyKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_KEY))

	order := p.MaybeOrderKeyword()
	conflictClause := p.MaybeConflictClause()

	var autoincrement *ast.Keyword = nil
	if p.Current().Kind == token.TokenKind_Keyword_AUTOINCREMENT {
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

func (p *SqliteParser) ColumnConstraint_NotNull(constraintName *ast.ConstraintName) *ast.ColumnConstraint_NotNull {
	p.PushParseContext("not null column constraint")
	defer p.PopParseContext()

	notKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_NOT))
	nullKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_NULL))

	conflictClause := p.MaybeConflictClause()

	return ast.MakeColumnConstraintNotNull(
		constraintName,
		notKeyword,
		nullKeyword,
		conflictClause,
	)
}

func (p *SqliteParser) ColumnConstraint_Default(constraintName *ast.ConstraintName) *ast.ColumnConstraint_Default {
	p.PushParseContext("default column constraint")
	defer p.PopParseContext()

	defaultKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_DEFAULT))

	if p.Current().Kind == '(' {
		p.Advance()
		expr := p.Expr(0)
		p.Expect(')')
		return ast.MakeColumnConstraintDefault(constraintName, defaultKeyword, expr)
	}

	lit, err := ast.TokenToLiteral(p.Current())
	if err != nil {
		rep := report.NewReport("parse error").
			WithLocation(p.Current().FileLoc).
			WithNotes("expected '(expr)' or literal value for DEFAULT column constraint)").
			WithLabels(report.LabelFromToken(p.Current(), "here"))
		p.ReportError(rep)

		// if the next token ahead is the start of a new column constraint or the end of column/table def
		// then we can skip the current token with the reported error above
		tok := p.Current()
		if isColumnConstraintStartingToken(p.Peeked()) || p.Peeked().Kind == ',' || p.Peeked().Kind == ')' {
			p.Advance()
		}
		lit = ast.MakeParseError(err, tok)
	}

	return ast.MakeColumnConstraintDefault(constraintName, defaultKeyword, lit)
}

func (p *SqliteParser) ColumnConstraint_Check(constraintName *ast.ConstraintName) *ast.ColumnConstraint_Check {
	p.PushParseContext("column constraint check")
	defer p.PopParseContext()

	checkKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CHECK))
	expr := p.Expr(0)
	return ast.MakeColumnConstraintCheck(constraintName, checkKeyword, expr)
}

func (p *SqliteParser) MaybeConstraintName() *ast.ConstraintName {
	p.PushParseContext("constraint name")
	defer p.PopParseContext()

	// we may or may not have CONSTRAINT keyword here so have a look-y and check-y
	if p.Current().Kind != token.TokenKind_Keyword_CONSTRAINT {
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
	if p.Current().Kind != token.TokenKind_Keyword_ON {
		return nil
	}
	onKeyword := ast.Keyword(p.Current())
	p.Advance()

	conflictKeyword := ast.Keyword(p.Expect(token.TokenKind_Keyword_CONFLICT))

	switch p.Current().Kind {
	case token.TokenKind_Keyword_ROLLBACK,
		token.TokenKind_Keyword_ABORT,
		token.TokenKind_Keyword_FAIL,
		token.TokenKind_Keyword_IGNORE,
		token.TokenKind_Keyword_REPLACE:
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
					WithLocation(p.Current().FileLoc).
					WithLabels(
						report.LabelFromToken(p.Current(), "expected 'rollback', 'abort', 'fail', 'ignore' or 'replace'"),
					).
					WithNotes(fmt.Sprintf("got '%s'", p.Current().Text)),
			)
			return nil
		}
	}
}

func (p *SqliteParser) MaybeOrderKeyword() *ast.Keyword {
	switch p.Current().Kind {
	case token.TokenKind_Keyword_ASC:
		fallthrough
	case token.TokenKind_Keyword_DESC:
		return ast.MakeKeyword(p.Current())
	default:
		return nil
	}
}

func (p *SqliteParser) Expr(minBindingPower int) ast.Expr {
	return p.Parser.Expr(minBindingPower, p)
}

func (p *SqliteParser) Term() ast.Expr {
	switch p.Current().Kind {
	case token.TokenKind_StringLiteral:
		result := &ast.LiteralString{
			Token: p.Current(),
			Value: p.Current().Text,
		}
		p.Advance()
		return result
	case token.TokenKind_Identifier:
		result := ast.Identifier(p.Current())
		p.Advance()
		return &result
	default:
		panic("expression type not handled")
	}
}

func (p *SqliteParser) OperatorBindingPower(tok token.Token) (bp ast.BindingPower, found bool) {
	switch tok.Kind {
	case '+':
		return ast.BindingPower{L: 100, R: 101}, true
	default:
		return ast.BindingPower{}, false
	}
}
