package parser

import (
	"fmt"
	"strconv"
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
	ParseContext datastructures.Stack[ParseContext]
	CurrentToken tik.Token
	PeekedToken  tik.Token

	Lexer *luther.Lexer

	Errors   map[tik.TextRange]report.Report
	Warnings map[tik.TextRange]report.Report
}

func NewParser(lexer *luther.Lexer) *Parser {

	result := &Parser{
		Lexer:        lexer,
		Errors:       map[tik.TextRange]report.Report{},
		Warnings:     map[tik.TextRange]report.Report{},
		ParseContext: datastructures.Stack[ParseContext]{},
	}

	result.CurrentToken = lexer.NextToken()
	result.PeekedToken = lexer.PeekToken()

	return result
}

func (p *Parser) ReportError(report *report.Report) {

	if _, has := p.Errors[p.CurrentToken.SourceRange]; has {
		panic(report)
	}

	p.Errors[p.CurrentToken.SourceRange] = *report
}

func (p *Parser) ReportWarning(report *report.Report) {
	p.Warnings[p.CurrentToken.SourceRange] = *report
}

func (p *Parser) ParseContextsToLabels() []report.Label {
	result := []report.Label{}

	for _, item := range p.ParseContext.Data {
		result = append(result, report.Label{
			Source: p.Lexer.SourceCode,
			Range: tik.TextRange{
				Start: item.StartingToken.SourceRange.Start,
				End:   item.EndingToken.SourceRange.End,
			},
			Note: item.Name,
		})
	}

	return result
}

func (p *Parser) Advance() {
	p.CurrentToken = p.Lexer.NextToken()
	p.PeekedToken = p.Lexer.PeekToken()
}

func (p *Parser) Expect(kind tik.TokenKind) tik.Token {
	if p.CurrentToken.Kind == kind {
		token := p.CurrentToken
		p.Advance()
		return token
	}

	if err, ok := p.Errors[p.CurrentToken.SourceRange]; ok {
		panic(err)
	}

	parseContext, ok := p.ParseContext.Top()

	if ok {
		p.ReportError(
			report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   fmt.Sprintf("expected '%s' got '%s'", kind.DebugString(), p.CurrentToken.DebugString()),
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
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   fmt.Sprintf("expected '%s' got '%s'", kind.DebugString(), p.CurrentToken.DebugString()),
					},
				}),
		)
	}

	if p.PeekedToken.Kind == kind {
		builder := strings.Builder{}
		builder.WriteString(p.CurrentToken.String())
		p.Advance()
		builder.WriteString(p.CurrentToken.LeadingTrivia)
		p.CurrentToken.LeadingTrivia = builder.String()
		realToken := p.CurrentToken
		p.Advance() // Consume '('
		return realToken
	}

	costSynthesis := tik.Token{Kind: kind}.InsertionCost()
	costDeletion := p.CurrentToken.DeletionCost()

	if costSynthesis <= costDeletion {
		// p.Advance()
		return tik.Token{
			Kind: kind,
		}
	}

	builder := strings.Builder{}
	builder.WriteString(p.CurrentToken.String())
	p.Advance()
	builder.WriteString(p.CurrentToken.LeadingTrivia)
	p.CurrentToken.LeadingTrivia = builder.String()
	return p.CurrentToken
}

func (p *Parser) synchronize() {
	for !p.Lexer.Eof() {
		switch p.CurrentToken.Kind {
		case ';':
			p.Advance()
			return
		default:
			p.Advance()
		}
	}
}

func (p *Parser) PushParseContext(name string) {
	p.ParseContext.Push(
		ParseContext{
			Name:          name,
			StartingToken: p.CurrentToken,
			EndingToken:   p.CurrentToken,
		},
	)
}

func (p *Parser) PopParseContext() {
	if len(p.ParseContext.Data) > 0 {
		p.ParseContext.Data[len(p.ParseContext.Data)-1].EndingToken = p.CurrentToken
	}
	p.ParseContext.Pop()
}

func (p *Parser) Statements() []ast.Statement {

	statements := []ast.Statement{}

	for !p.Lexer.Eof() {
		func() {
			defer func() {
				if r := recover(); r != nil {
					p.synchronize()
				}
			}()

			// 3. Try to parse normally
			statement := p.Statement()
			statements = append(statements, statement)

			// 4. Expect the terminator
			// If this fails/panics, the defer block above handles it too.
			p.Expect(';')
		}()
	}

	return statements
}

func (p *Parser) Statement() ast.Statement {

	p.PushParseContext("statement")
	defer p.PopParseContext()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_PRAMGA:
		return p.PragmaStatement()
	case tik.TokenKind_Keyword_CREATE:
		return p.CreateStatement()
	case tik.TokenKind_Keyword_BEGIN:
		return p.BeginStatement()
	case tik.TokenKind_Keyword_COMMIT:
		p.Advance()
		return &ast.CommitTransaction{}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   fmt.Sprintf("unknown token at start of sql statement '%s'", p.CurrentToken.DebugString()),
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) BeginStatement() ast.Statement {
	p.Expect(tik.TokenKind_Keyword_BEGIN)
	p.Expect(tik.TokenKind_Keyword_TRANSACTION)
	return &ast.BeginTransaction{}
}

func (p *Parser) PragmaStatement() ast.Statement {

	p.PushParseContext("pragma statement")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_PRAMGA)
	pragmaIdentifier := p.CatalogObjectIdentifier()

	switch token := p.CurrentToken; token.Kind {
	case '=':
		{
			p.Advance()
			pragmaValue := p.PragmaValue()
			return &ast.Pragma{
				Name:  pragmaIdentifier,
				Value: pragmaValue,
			}
		}
	case '(':
		{
			p.Advance()
			pragmaValue := p.PragmaValue()
			p.Expect(')')
			return &ast.Pragma{
				Name:  pragmaIdentifier,
				Value: pragmaValue,
			}
		}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unknown token after pragma identifier",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) PragmaValue() ast.PragmaValue {

	p.PushParseContext("pragma value")
	defer p.PopParseContext()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_DecimalNumericLiteral:
		p.Advance()
		return &ast.LiteralNumber{
			Token: token,
			Value: p.TokenToNumber(token),
		}
	case tik.TokenKind_Identifier:
		p.Advance()
		result := ast.Identifier(token)
		return &result
	case tik.TokenKind_StringLiteral:
		p.Advance()
		return &ast.LiteralString{
			Token: token,
			Value: token.Text,
		}
	case tik.TokenKind_Keyword_TRUE, tik.TokenKind_Keyword_FALSE, tik.TokenKind_Keyword_ON:
		p.Advance()
		return &ast.LiteralBoolean{
			Token: token,
			Value: p.TokenToBoolean(token),
		}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unknown token for pragma value",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) CreateStatement() ast.Statement {

	p.PushParseContext("statement")
	defer p.PopParseContext()

	createKeyword := ast.MakeKeyword(
		p.Expect(tik.TokenKind_Keyword_CREATE),
	)

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(createKeyword, false)
	case tik.TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(false)
	case tik.TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(false)
	case tik.TokenKind_Keyword_INDEX:
		return p.CreateIndexStatement(createKeyword, false)
	case tik.TokenKind_Keyword_UNIQUE:
		return p.CreateIndexStatement(createKeyword, true)
	case tik.TokenKind_Keyword_VIRTUAL:
		return p.CreateVirtualTableStatement()
	case tik.TokenKind_Keyword_TEMPORARY:
		return p.CreateTemporaryStatement(createKeyword)
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unknown token for create statement",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) CreateTemporaryStatement(createKeyword *ast.Keyword) ast.Statement {

	p.PushParseContext("temporary")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_TEMPORARY)

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(createKeyword, true)
	case tik.TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(true)
	case tik.TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(true)
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unknown token for create temporary statement",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) CreateViewStatement(temporary bool) ast.Statement {

	p.PushParseContext("create view statement")
	defer p.PopParseContext()

	if temporary {
		p.Expect(tik.TokenKind_Keyword_TEMPORARY)
	}

	p.Expect(tik.TokenKind_Keyword_VIEW)

	ifnotexists := p.MaybeIfNotExists()

	viewIdentifier := p.CatalogObjectIdentifier()

	columnNames := []ast.Identifier{}
	if p.CurrentToken.Kind == '(' {

	ColumnNamesLoop:
		for !p.Lexer.Eof() {
			switch p.CurrentToken.Kind {
			case ',':
				p.Advance()
				continue
			case ')':
				break ColumnNamesLoop
			default:
				columnName := p.Identifier()
				columnNames = append(columnNames, columnName)
			}
		}

		p.Expect(')')
	}

	p.Expect(tik.TokenKind_Keyword_AS)

	selectStmt := p.SelectStatement()

	return &ast.CreateView{
		IfNotExists:    ifnotexists,
		Columns:        columnNames,
		ViewIdentifier: viewIdentifier,
		AsSelect:       selectStmt,
	}
}

func (p *Parser) CreateTriggerStatement(temporary bool) (result ast.Statement) {

	p.PushParseContext("create trigger statement")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_TRIGGER)

	for p.CurrentToken.Kind != tik.TokenKind_Keyword_BEGIN {
		p.Advance()
	}
	p.Advance()
	for p.CurrentToken.Kind != ';' {
		p.Advance()
	}
	p.Advance()
	for p.CurrentToken.Kind != tik.TokenKind_Keyword_END {
		p.Advance()
	}
	p.Advance()

	return &ast.CreateTrigger{}
}

func (p *Parser) SelectStatement() ast.Statement {

	p.PushParseContext("select statement")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_SELECT)

	for !p.Lexer.Eof() {
		if p.CurrentToken.Kind == ';' {
			break
		}
		p.Advance()
	}

	return &ast.Select{}
}

func (p *Parser) CreateTableStatement(createKeyword *ast.Keyword, isTemporary bool) ast.Statement {

	p.PushParseContext("create table statement")
	defer p.PopParseContext()

	var temporary *ast.Keyword = nil
	if isTemporary {
		temporary = ast.MakeKeyword(p.Expect(tik.TokenKind_Keyword_TEMPORARY))
	}

	tableKeyword := ast.MakeKeyword(p.Expect(tik.TokenKind_Keyword_TABLE))

	ifnotexists := p.MaybeIfNotExists()
	tableIdentifier := p.CatalogObjectIdentifier()
	tableDefinition := p.TableDefinition()
	tableOptions := p.TableOptions()

	return &ast.CreateTable{
		CreateKeyword:   *createKeyword,
		Temporary:       temporary,
		TableKeyword:    *tableKeyword,
		IfNotExist:      ifnotexists,
		TableIdentifier: &tableIdentifier,
		TableDefinition: &tableDefinition,
		TableOptions:    tableOptions,
	}
}

func (p *Parser) CreateVirtualTableStatement() ast.Statement {

	p.PushParseContext("create virtual statement")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_VIRTUAL)

	p.Expect(tik.TokenKind_Keyword_TABLE)

	ifnotexists := p.MaybeIfNotExists()

	tableIdentifier := p.CatalogObjectIdentifier()

	p.Expect(tik.TokenKind_Keyword_USING)

	moduleName := p.Identifier()

	args := []string{}
	if p.CurrentToken.Kind == '(' {
		p.Advance()
		str := string("")
	ModuleArgsLoop:
		for !p.Lexer.Eof() {
			switch token := p.CurrentToken; token.Kind {
			case ',':
				p.Advance()
				args = append(args, str)
				str = string("")
				continue
			case ')':
				args = append(args, str)
				str = string("")
				break ModuleArgsLoop
			default:
				p.Advance()
				str = strings.Join([]string{str, token.String()}, "")
			}
		}

		p.Expect(')')
	}

	return &ast.CreateVirtualTable{
		IfNotExist:      ifnotexists,
		TableIdentifier: tableIdentifier,
		ModuleName:      moduleName,
		ModuleArgs:      args,
	}
}

func (p *Parser) CreateIndexStatement(createKeyword *ast.Keyword, isUnique bool) ast.Statement {

	p.PushParseContext("create index statement")
	defer p.PopParseContext()

	var unique *ast.Keyword = nil
	if isUnique {
		unique = ast.MakeKeyword(
			p.Expect(tik.TokenKind_Keyword_UNIQUE),
		)
	}

	indexKeyword := ast.MakeKeyword(
		p.Expect(tik.TokenKind_Keyword_INDEX),
	)

	ifnotexists := p.MaybeIfNotExists()

	indexIdentifier := p.CatalogObjectIdentifier()

	p.Expect(tik.TokenKind_Keyword_ON)

	tableName := p.CatalogObjectIdentifier()

	p.Expect('(')

	indexedColumns := []ast.IndexedColumn{}
IndexedColumnsLoop:
	for !p.Lexer.Eof() {
		switch p.CurrentToken.Kind {
		case ',':
			p.Advance()
			continue
		case ')':
			break IndexedColumnsLoop
		default:
			indexedColumn := p.IndexedColumn(true)
			indexedColumns = append(indexedColumns, indexedColumn)
		}
	}

	p.Expect(')')

	var whereExpr ast.Expr = nil
	if p.CurrentToken.Kind == tik.TokenKind_Keyword_WHERE {
		p.Advance()
		whereExpr = p.Expr(0)
	}

	return &ast.CreateIndex{
		CreateKeyword:   *createKeyword,
		IndexKeyword:    *indexKeyword,
		Unique:          unique,
		IfNotExists:     ifnotexists,
		IndexIdentifier: indexIdentifier,
		OnTable:         tableName,
		IndexedColumns:  indexedColumns,
		WhereExpr:       whereExpr,
	}
}

func (p *Parser) IndexedColumn(allowExpressions bool) ast.IndexedColumn {

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
	order := p.MaybeOrderBy()

	return ast.IndexedColumn{
		Subject:   expr,
		Collation: collation,
		Order:     order,
	}
}

func (p *Parser) MaybeIfNotExists() *ast.IfNotExists {

	p.PushParseContext("if not exists")
	defer p.PopParseContext()

	var if_ *ast.Keyword = nil

	if token := p.CurrentToken; token.Kind != tik.TokenKind_Keyword_IF {
		return nil
	} else {
		p.Advance()
		if_ = ast.MakeKeyword(token)
	}

	not := ast.MakeKeyword(p.Expect(tik.TokenKind_Keyword_NOT))
	exists := ast.MakeKeyword(p.Expect(tik.TokenKind_Keyword_EXISTS))

	return &ast.IfNotExists{
		If:     *if_,
		Not:    *not,
		Exists: *exists,
	}
}

func (p *Parser) MaybeCollation() *ast.Collation {
	if p.CurrentToken.Kind != tik.TokenKind_Keyword_COLLATE {
		return nil
	}
	return ast.MakeCollation(
		ast.Keyword(p.CurrentToken),
		p.Identifier(),
	)
}

func (p *Parser) CatalogObjectIdentifier() ast.CatalogObjectIdentifier {

	p.PushParseContext("catalog object identifier")
	defer p.PopParseContext()

	schemaOrTable := p.Identifier()

	if p.CurrentToken.Kind != tik.TokenKind_Period {
		return ast.CatalogObjectIdentifier{
			SchemaName: nil,
			ObjectName: schemaOrTable,
		}
	}
	p.Advance()

	table := p.Identifier()

	return ast.CatalogObjectIdentifier{
		SchemaName: &schemaOrTable,
		ObjectName: table,
	}
}

func (p *Parser) TableDefinition() ast.TableDefinition {

	p.PushParseContext("table definition")
	defer p.PopParseContext()

	p.Expect('(')

	columnDefs := p.ColumnDefinitions()
	tableConstraints := p.TableConstraints()

	p.Expect(')')

	return ast.TableDefinition{
		ColumnDefinitions: columnDefs,
		TableConstraints:  tableConstraints,
	}
}

func (p *Parser) ColumnDefinitions() []ast.ColumnDefinition {

	p.PushParseContext("column definitions")
	defer p.PopParseContext()

	definitions := []ast.ColumnDefinition{}

ColumnDefinitionsLoop:
	for !p.Lexer.Eof() {
		switch token := p.CurrentToken; {
		case token.Kind == ',':
			p.Advance()
			continue
		case token.Kind == ')':
			break ColumnDefinitionsLoop
		case isConstraintKeyword(token):
			break ColumnDefinitionsLoop
		default:
			columnDef := p.ColumnDefinition()
			definitions = append(definitions, columnDef)
		}
	}

	return definitions
}

func isConstraintKeyword(token tik.Token) bool {
	_, ok := tik.ConstaintKeywords[token.Kind]
	return ok
}

func (p *Parser) ColumnDefinition() ast.ColumnDefinition {

	p.PushParseContext("column definition")
	defer p.PopParseContext()

	columnName := p.Identifier()
	typeName := p.TypeName()
	columnConstraints := p.ColumnConstraints()

	return ast.ColumnDefinition{
		ColumnName:        columnName,
		TypeName:          typeName,
		ColumnConstraints: columnConstraints,
	}
}

func (p *Parser) ColumnConstraints() []ast.ColumnConstraint {

	p.PushParseContext("column constraints")
	defer p.PopParseContext()

	result := []ast.ColumnConstraint{}

ColumnConstraintsLoop:
	for !p.Lexer.Eof() {
		switch token := p.CurrentToken; token.Kind {
		case ',', ')':
			break ColumnConstraintsLoop
		default:
			columnConstraint := p.ColumnConstraint()
			result = append(result, columnConstraint)
		}
	}

	return result
}

func (p *Parser) ColumnConstraint() ast.ColumnConstraint {

	p.PushParseContext("column constraint")
	defer p.PopParseContext()

	constraintName := p.MaybeConstraintName()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_PRIMARY:
		return p.ColumnConstraint_PrimaryKey(constraintName)
	case tik.TokenKind_Keyword_NOT:
		return p.ColumnConstraint_NotNull(constraintName)
	case tik.TokenKind_Keyword_DEFAULT:
		return p.ColumnConstraint_Default(constraintName)
	case tik.TokenKind_Keyword_UNIQUE:
		return p.ColumnConstraint_Unique(constraintName)
	case tik.TokenKind_Keyword_COLLATE:
		return p.ColumnConstraint_Collate(constraintName)
	case tik.TokenKind_Keyword_CHECK:
		return p.ColumnConstraint_Check(constraintName)
	case tik.TokenKind_Keyword_AS:
		return p.ColumnConstraint_Generated(constraintName)
	case tik.TokenKind_Keyword_GENERATED:
		return p.ColumnConstraint_Generated(constraintName)
	default:
		{
			p.ReportError(
				report.
					NewReport("parse error").
					WithLabels([]report.Label{
						{
							Source: p.CurrentToken.SourceCode,
							Range:  p.CurrentToken.SourceRange,
							Note:   "expected beginning of column constraint",
						},
					}),
			)
			return nil
		}
	}
}

func (p *Parser) ColumnConstraint_PrimaryKey(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("primary key column constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_PRIMARY)
	p.Expect(tik.TokenKind_Keyword_KEY)

	orderBy := p.MaybeOrderBy()
	conflictclause := p.MaybeConflictClause()

	var autoincrement *ast.Keyword = nil
	if tok, found := p.MaybeTokenKind(tik.TokenKind_Keyword_AUTOINCREMENT); found {
		autoincrement = ast.MakeKeyword(tok)
	}

	return &ast.ColumnConstraint_PrimaryKey{
		Name:           constraintName,
		ConflictClause: conflictclause,
		Order:          orderBy,
		AutoIncrement:  autoincrement,
	}
}

func (p *Parser) MaybeTokenKind(kind tik.TokenKind) (tik.Token, bool) {
	if token := p.CurrentToken; token.Kind == kind {
		p.Advance()
		return token, true
	}
	return tik.Token{}, false
}

func (p *Parser) ColumnConstraint_NotNull(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("not null column constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_NOT)
	p.Expect(tik.TokenKind_Keyword_NULL)

	return &ast.ColumnConstraint_NotNull{}
}

func (p *Parser) ColumnConstraint_Default(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("default column constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_DEFAULT)

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_StringLiteral:
		p.Advance()
		return &ast.ColumnConstraint_Default{
			Default: &ast.LiteralString{
				Token: token,
				Value: token.Text,
			},
		}
	case tik.TokenKind_DecimalNumericLiteral, tik.TokenKind_BinaryNumericLiteral, tik.TokenKind_HexNumericLiteral, tik.TokenKind_OctalNumericLiteral:
		p.Advance()
		return &ast.ColumnConstraint_Default{
			Default: &ast.LiteralNumber{
				Token: token,
				Value: p.TokenToNumber(token),
			},
		}
	case tik.TokenKind_Keyword_TRUE, tik.TokenKind_Keyword_FALSE:
		p.Advance()
		return &ast.ColumnConstraint_Default{
			Default: &ast.LiteralBoolean{
				Token: token,
				Value: p.TokenToBoolean(token),
			},
		}
	case tik.TokenKind_Identifier:
		p.Advance()
		ident := ast.Identifier(token)
		return &ast.ColumnConstraint_Default{
			Default: &ident,
		}
	default:
		panic("not implemented")
	}
}

func (p *Parser) TokenToBoolean(token tik.Token) ast.Boolean {
	switch token.Kind {
	case tik.TokenKind_Keyword_TRUE:
		return ast.Boolean(true)
	case tik.TokenKind_Keyword_FALSE:
		return ast.Boolean(false)
	case tik.TokenKind_Keyword_ON:
		return ast.Boolean(true)
	default:
		panic("unreachable")
	}
}

func (p *Parser) TokenToNumber(token tik.Token) ast.AstNode {
	switch token.Kind {
	case tik.TokenKind_HexNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 16, 64)
		if err != nil {
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "unable to parse hex numeric literal to uint",
					},
				})
			p.ReportError(err)
			return nil
		}
		return ast.Integer(value)
	case tik.TokenKind_OctalNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 8, 64)
		if err != nil {
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "unable to parse octal numeric literal to uint",
					},
				})
			p.ReportError(err)
			return nil
		}
		return ast.Integer(value)
	case tik.TokenKind_BinaryNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 2, 64)
		if err != nil {
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "unable to parse binary numeric literal to uint",
					},
				})
			p.ReportError(err)
			return nil
		}
		return ast.Integer(value)
	case tik.TokenKind_DecimalNumericLiteral:
		value, err := strconv.ParseFloat(token.Text, 64)
		if err != nil {
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "unable to parse decimal numeric literal to float",
					},
				})
			p.ReportError(err)
			return nil
		}
		return ast.Float(value)
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unable to parse token as number",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) ColumnConstraint_Unique(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("unique column constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_UNIQUE)

	return &ast.ColumnConstraint_Unique{
		Name: constraintName,
	}
}

func (p *Parser) ColumnConstraint_Collate(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("collate column constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_COLLATE)

	collationName := p.Identifier()

	return &ast.ColumnConstraint_Collate{
		Name:    constraintName,
		Collate: collationName,
	}
}

func (p *Parser) ColumnConstraint_Generated(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("generated column")
	defer p.PopParseContext()

	if p.CurrentToken.Kind == tik.TokenKind_Keyword_GENERATED {
		p.Advance()
		p.Expect(tik.TokenKind_Keyword_ALWAYS)
	}

	p.Expect(tik.TokenKind_Keyword_AS)
	p.Expect('(')
	expr := p.Expr(0)
	p.Expect(')')

	storage := p.GeneratedColumnStorage()

	return &ast.ColumnConstraint_Generated{
		Name:    constraintName,
		As:      expr,
		Storage: storage,
	}
}

func (p *Parser) GeneratedColumnStorage() ast.AstNode {
	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_VIRTUAL:
		p.Advance()
		return ast.MakeKeyword(token)
	case tik.TokenKind_Keyword_STORED:
		p.Advance()
		return ast.MakeKeyword(token)
	default:
		return nil
	}
}

func (p *Parser) MaybeConstraintName() *ast.ConstraintName {
	p.PushParseContext("constraint name")
	defer p.PopParseContext()

	// we may or may not have a constraint keyword here so peek and check
	if p.CurrentToken.Kind != tik.TokenKind_Keyword_CONSTRAINT {
		return nil
	}
	constraintKeyword := ast.Keyword(p.CurrentToken)
	p.Advance()

	name := p.Identifier()

	return &ast.ConstraintName{
		ConstraintKeyword: constraintKeyword,
		Name:              name,
	}
}

func (p *Parser) TableConstraints() []ast.TableConstraint {
	p.PushParseContext("table constraints")
	defer p.PopParseContext()

	result := []ast.TableConstraint{}

TableConstraintsLoop:
	for !p.Lexer.Eof() {
		switch p.CurrentToken.Kind {
		case ')':
			break TableConstraintsLoop
		case ',':
			p.Advance()
			continue
		default:
			tableConstraint := p.TableConstraint()
			result = append(result, tableConstraint)
		}
	}

	return result
}

func (p *Parser) TableConstraint() ast.TableConstraint {

	p.PushParseContext("table constraint")
	defer p.PopParseContext()

	constraintName := p.MaybeConstraintName()
	if constraintName == nil {
		p.ReportWarning(
			report.NewReport("warning").
				WithMessage("unnamed table constraint").
				WithLabels([]report.Label{
					{
						Source: p.Lexer.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "CONSTRAINT constraint_name",
					},
				}).
				WithNotes([]string{
					"by adding a constraint name, we can detect changes of table constraints, and migrate them appropriately.",
				}),
		)
	}

	switch token := p.CurrentToken; token.Kind {
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
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "unexpected token for table constraint",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) TableConstraint_PrimaryKey(constraintName *ast.ConstraintName) ast.TableConstraint {

	var autoincrement *ast.Keyword = nil

	p.PushParseContext("primary key table constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_PRIMARY)
	p.Expect(tik.TokenKind_Keyword_KEY)
	p.Expect('(')

	indexedCols := []ast.IndexedColumn{}

	// take the first one manually incase it is followed by autoincrement
	indexedCol := p.IndexedColumn(false)
	indexedCols = append(indexedCols, indexedCol)

	if token, ok := p.MaybeTokenKind(tik.TokenKind_Keyword_AUTOINCREMENT); ok {
		autoincrement = ast.MakeKeyword(token)
	}

	if autoincrement == nil {
	IndexedColumnsLoop:
		for !p.Lexer.Eof() {
			switch p.CurrentToken.Kind {
			case ',':
				p.Advance()
				continue
			case ')':
				break IndexedColumnsLoop
			default:
				indexedCol := p.IndexedColumn(false)
				indexedCols = append(indexedCols, indexedCol)
			}
		}
	}

	p.Expect(')')

	conflictClause := p.MaybeConflictClause()

	tableConstraint := &ast.TableConstraint_PrimaryKey{
		Name:           constraintName,
		IndexedColumns: indexedCols,
		ConflictClause: conflictClause,
		AutoIncrement:  autoincrement,
	}

	return tableConstraint
}

func (p *Parser) TableConstraint_ForeignKey(constraintName *ast.ConstraintName) ast.TableConstraint {

	p.PushParseContext("foreign key table constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_FOREIGN)
	p.Expect(tik.TokenKind_Keyword_KEY)
	p.Expect('(')

	columnNames := []ast.Identifier{}

ColumnNamesLoop:
	for !p.Lexer.Eof() {
		switch p.CurrentToken.Kind {
		case ',':
			p.Advance()
			continue
		case ')':
			break ColumnNamesLoop
		default:
			columnName := p.Identifier()
			columnNames = append(columnNames, columnName)
		}
	}

	p.Expect(')')

	fkClause := p.ForeignKeyClause()

	return &ast.TableConstraint_ForeignKey{
		Name:     constraintName,
		Columns:  columnNames,
		FkClause: fkClause,
	}
}

func (p *Parser) ForeignKeyClause() ast.ForeignKeyClause {

	p.PushParseContext("foreign key clause")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_REFERENCES)
	foreignTable := p.CatalogObjectIdentifier()

	foreignColumns := []ast.Identifier{}

	if p.CurrentToken.Kind == '(' {
		p.Advance()
	ForeignColumnsLoop:
		for !p.Lexer.Eof() {
			switch p.CurrentToken.Kind {
			case ',':
				p.Advance()
				continue
			case ')':
				break ForeignColumnsLoop
			default:
				columnName := p.Identifier()
				foreignColumns = append(foreignColumns, columnName)
			}
		}

		p.Expect(')')
	}

	actions := []ast.ForeignKeyActionTrigger{}
	var matchName *ast.Identifier = nil
	var deferrable *ast.ForeignKeyDeferrable = nil

ForeignKeyModifiersLoop:
	for !p.Lexer.Eof() {
		switch p.CurrentToken.Kind {
		case tik.TokenKind_Keyword_ON:
			action := p.ForeignKeyActionTrigger()
			actions = append(actions, action)
			continue
		case tik.TokenKind_Keyword_MATCH:
			matchNameIdent := p.Identifier()
			matchName = &matchNameIdent
			continue
		case tik.TokenKind_Keyword_NOT:
			deferrable = p.ForeignKeyDeferrable()
			continue
		case tik.TokenKind_Keyword_DEFERRABLE:
			deferrable = p.ForeignKeyDeferrable()
			continue
		default:
			break ForeignKeyModifiersLoop
		}
	}

	return ast.ForeignKeyClause{
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
		Actions:        actions,
		MatchName:      matchName,
		Deferrable:     deferrable,
	}
}

func (p *Parser) ForeignKeyActionTrigger() ast.ForeignKeyActionTrigger {

	p.PushParseContext("foreign key action trigger")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_ON)

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_DELETE:
		p.Advance()
		return &ast.OnDelete{
			Action: p.ForeignKeyAction(),
		}
	case tik.TokenKind_Keyword_UPDATE:
		p.Advance()
		return &ast.OnUpdate{
			Action: p.ForeignKeyAction(),
		}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "expected action trigger keyword 'delete' or 'update' for fk action",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) ForeignKeyAction() ast.ForeignKeyAction {
	p.PushParseContext("foreign key action")
	defer p.PopParseContext()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_CASCADE:
		p.Advance()
		return &ast.Cascade{}
	case tik.TokenKind_Keyword_RESTRICT:
		p.Advance()
		return &ast.Restrict{}
	case tik.TokenKind_Keyword_NO:
		p.Advance()
		p.Expect(tik.TokenKind_Keyword_ACTION)
		return &ast.NoAction{}
	case tik.TokenKind_Keyword_SET:
		p.Advance()
		switch token := p.CurrentToken; token.Kind {
		case tik.TokenKind_Keyword_DEFAULT:
			p.Advance()
			return &ast.SetDefault{}
		case tik.TokenKind_Keyword_NULL:
			p.Advance()
			return &ast.SetNull{}
		default:
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "expected keyword 'default' or 'null' for fk action 'set'",
					},
				})
			p.ReportError(err)
			return nil
		}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "expected fk action method 'cascade', 'restrict', 'no action', 'set default' or 'set null'",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) ForeignKeyDeferrable() *ast.ForeignKeyDeferrable {

	p.PushParseContext("foreign key deferrable")
	defer p.PopParseContext()

	var not *ast.Keyword = nil
	if p.CurrentToken.Kind == tik.TokenKind_Keyword_NOT {
		not = ast.MakeKeyword(p.CurrentToken)
		p.Advance()
	}

	p.Expect(tik.TokenKind_Keyword_DEFERRABLE)

	var initially *ast.Keyword = nil
	var deferrable *ast.Keyword = nil
	if p.CurrentToken.Kind == tik.TokenKind_Keyword_INITIALLY {
		initially = ast.MakeKeyword(p.CurrentToken)
		p.Advance()
		switch token := p.CurrentToken; token.Kind {
		case tik.TokenKind_Keyword_IMMEDIATE:
			deferrable = ast.MakeKeyword(token)
			p.Advance()
		case tik.TokenKind_Keyword_DEFERRED:
			p.Advance()
			deferrable = ast.MakeKeyword(token)
		default:
			err := report.
				NewReport("parse error").
				WithLabels([]report.Label{
					{
						Source: p.CurrentToken.SourceCode,
						Range:  p.CurrentToken.SourceRange,
						Note:   "expected deferrable keyword 'immediate' or 'deferred' after 'initally'",
					},
				})
			p.ReportError(err)
			return nil
		}
	}

	return &ast.ForeignKeyDeferrable{
		Not:        not,
		Initially:  initially,
		Deferrable: deferrable,
	}
}

func (p *Parser) TableOptions() *ast.TableOptions {

	p.PushParseContext("table options")
	defer p.PopParseContext()

	var strict *ast.Keyword = nil
	var withoutRowId *ast.WithoutRowId = nil

TableOptionsLoop:
	for !p.Lexer.Eof() {
		switch token := p.CurrentToken; token.Kind {
		case tik.TokenKind_Keyword_STRICT:
			p.Advance()
			strict = ast.MakeKeyword(token)
			continue
		case tik.TokenKind_Keyword_WITHOUT:
			withoutRowId = &ast.WithoutRowId{
				Without: ast.Keyword(token),
			}
			p.Advance()
			p.Expect(tik.TokenKind_Keyword_ROWID)
		default:
			break TableOptionsLoop
		}
	}

	return &ast.TableOptions{
		Strict:       strict,
		WithoutRowId: withoutRowId,
	}
}

func (p *Parser) MaybeConflictClause() *ast.ConflictClause {

	p.PushParseContext("conflict clause")
	defer p.PopParseContext()

	if p.CurrentToken.Kind != tik.TokenKind_Keyword_ON {
		return nil
	}
	onKeyword := ast.MakeKeyword(p.CurrentToken)
	p.Advance()

	conflictKeyword := p.Expect(tik.TokenKind_Keyword_CONFLICT)

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_ROLLBACK:
		fallthrough
	case tik.TokenKind_Keyword_ABORT:
		fallthrough
	case tik.TokenKind_Keyword_FAIL:
		fallthrough
	case tik.TokenKind_Keyword_IGNORE:
		fallthrough
	case tik.TokenKind_Keyword_REPLACE:
		p.Advance()
		return &ast.ConflictClause{
			OnKeyword:       *onKeyword,
			ConflictKeyword: *ast.MakeKeyword(conflictKeyword),
			Action:          *ast.MakeKeyword(token),
		}
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "expected conflict clause verb",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) TypeName() ast.TypeName {

	p.PushParseContext("type name")
	defer p.PopParseContext()

	ident := p.Identifier()

	return ast.TypeName{
		TypeName: ident,
	}
}

func (p *Parser) TableConstraint_Check(constraintName *ast.ConstraintName) ast.TableConstraint {

	p.PushParseContext("check constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_CHECK)
	p.Expect('(')

	expr := p.Expr(0)

	p.Expect(')')

	return &ast.TableConstraint_Check{
		Name: constraintName,
		Expr: expr,
	}
}

func (p *Parser) ColumnConstraint_Check(constraintName *ast.ConstraintName) ast.ColumnConstraint {

	p.PushParseContext("check constraint")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_CHECK)
	p.Expect('(')

	expr := p.Expr(0)

	p.Expect(')')

	return &ast.ColumnConstraint_Check{
		Name:  constraintName,
		Check: expr,
	}
}

func (p *Parser) MaybeOrderBy() *ast.Keyword {

	p.PushParseContext("order by")
	defer p.PopParseContext()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Keyword_ASC:
		fallthrough
	case tik.TokenKind_Keyword_DESC:
		p.Advance()
		result := ast.Keyword(token)
		return &result
	default:
		return nil
	}
}

func (p *Parser) Identifier() ast.Identifier {

	p.PushParseContext("identifier")
	defer p.PopParseContext()

	result := ast.Identifier(p.Expect(tik.TokenKind_Identifier))
	return result
}

func (p *Parser) nextOperatorBindingPower() (ast.BindingPower, bool) {
	switch token := p.CurrentToken; token.Kind {
	case '=':
		return ast.BindingPower{L: 40, R: 41}, true
	case '+':
		return ast.BindingPower{L: 60, R: 61}, true
	case '-':
		return ast.BindingPower{L: 60, R: 61}, true
	case '*':
		return ast.BindingPower{L: 120, R: 121}, true
	case '/':
		return ast.BindingPower{L: 120, R: 121}, true
	case tik.TokenKind_gte:
		return ast.BindingPower{L: 50, R: 51}, true
	case tik.TokenKind_Keyword_IN:
		return ast.BindingPower{L: 40, R: 41}, true
	default:
		return ast.BindingPower{}, false
	}
}

func (p *Parser) Expr(minBindingPower int) ast.Expr {

	p.PushParseContext("expression")
	defer p.PopParseContext()

	lhs := p.Nud()

	for !p.Lexer.Eof() {
		bindingPower, found := p.nextOperatorBindingPower()
		if !found {
			return lhs
		}
		if bindingPower.L < minBindingPower {
			break
		}

		opToken := p.CurrentToken
		p.Advance()
		rhs := p.Expr(bindingPower.R)

		binaryOp := &ast.BinaryOp{
			Operator: opToken,
			Lhs:      lhs,
			Rhs:      rhs,
		}
		lhs = binaryOp
	}

	return lhs
}

func (p *Parser) Identifier_Nud() ast.Expr {

	p.PushParseContext("identifier")
	defer p.PopParseContext()

	ident := ast.Identifier(p.CurrentToken)
	p.Advance()

	switch p.CurrentToken.Kind {
	case '(':
		p.Advance()
		arguments := p.ExprList(ast.ExprList{})
		p.Expect(')')
		return &ast.FunctionCall{
			Name: ident,
			Args: arguments,
		}
	case '.':
		p.Advance()
		tableOrColumn := p.Identifier()
		if p.CurrentToken.Kind != '.' {
			p.Advance()
			return &ast.ColumnName{
				Schema: nil,
				Table:  &ident,
				Column: tableOrColumn,
			}
		}

		column := p.Identifier()

		return &ast.ColumnName{
			Schema: &ident,
			Table:  &tableOrColumn,
			Column: column,
		}
	default:
		return &ident
	}
}

func (p *Parser) Nud() ast.Expr {
	p.PushParseContext("null denominator")
	defer p.PopParseContext()

	switch token := p.CurrentToken; token.Kind {
	case tik.TokenKind_Identifier:
		return p.Identifier_Nud()
	case tik.TokenKind_DecimalNumericLiteral, tik.TokenKind_BinaryNumericLiteral, tik.TokenKind_OctalNumericLiteral, tik.TokenKind_HexNumericLiteral:
		p.Advance()
		return &ast.LiteralNumber{
			Token: token,
			Value: p.TokenToNumber(token),
		}
	case tik.TokenKind_StringLiteral:
		p.Advance()
		return &ast.LiteralString{
			Token: token,
			Value: token.Text,
		}
	case tik.TokenKind_Keyword_TRUE, tik.TokenKind_Keyword_FALSE:
		p.Advance()
		return &ast.LiteralBoolean{
			Token: token,
			Value: p.TokenToBoolean(token),
		}
	case tik.TokenKind_Keyword_CASE:
		return p.CaseExpr()
	case '(':
		p.Advance()
		result := p.Expr(0)

		if p.CurrentToken.Kind == ',' {
			p.Advance()
			list := ast.ExprList{result}

			result = p.ExprList(list)
		}
		p.Expect(')')
		return result
	default:
		err := report.
			NewReport("parse error").
			WithLabels([]report.Label{
				{
					Source: p.CurrentToken.SourceCode,
					Range:  p.CurrentToken.SourceRange,
					Note:   "expected expression leaf",
				},
			})
		p.ReportError(err)
		return nil
	}
}

func (p *Parser) CaseExpr() ast.Expr {

	p.PushParseContext("case expression")
	defer p.PopParseContext()

	p.Expect(tik.TokenKind_Keyword_CASE)

	var operand ast.Expr = nil
	if p.CurrentToken.Kind != tik.TokenKind_Keyword_WHEN {
		operand = p.Expr(0)
	}

	cases := []ast.WhenThen{}

CasesLoop:
	for !p.Lexer.Eof() {
		p.Expect(tik.TokenKind_Keyword_WHEN)
		when := p.Expr(0)

		p.Expect(tik.TokenKind_Keyword_THEN)
		then := p.Expr(0)

		cases = append(cases, ast.WhenThen{
			When: when,
			Then: then,
		})

		switch p.CurrentToken.Kind {
		case tik.TokenKind_Keyword_ELSE, tik.TokenKind_Keyword_END:
			break CasesLoop
		}
	}

	var elseExpr ast.Expr = nil
	if p.CurrentToken.Kind == tik.TokenKind_Keyword_ELSE {
		p.Advance()
		elseExpr = p.Expr(0)
	}

	p.Expect(tik.TokenKind_Keyword_END)

	return &ast.CaseExpression{
		Operand: operand,
		Cases:   cases,
		Else:    elseExpr,
	}
}

func (p *Parser) ExprList(in ast.ExprList) ast.ExprList {

	p.PushParseContext("list of expressions")
	defer p.PopParseContext()

	result := in

ExprListLoop:
	for !p.Lexer.Eof() {

		switch p.CurrentToken.Kind {
		case ',':
			p.Advance()
			continue
		case ')':
			break ExprListLoop
		default:
			expr := p.Expr(0)
			result = append(result, expr)
		}
	}

	return result
}
