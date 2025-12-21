package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func unused[T any](value T) {}

type Parser struct {
	CurrentToken Token
	PeekedToken  Token
	tokenizer    *Tokenizer
}

func NewParser(tokenizer *Tokenizer) *Parser {

	result := &Parser{
		tokenizer: tokenizer,
	}

	result.CurrentToken = tokenizer.NextToken()
	result.PeekedToken = tokenizer.PeekToken()

	return result
}

func (p *Parser) Advance() {
	p.CurrentToken = p.tokenizer.NextToken()
	p.PeekedToken = p.tokenizer.PeekToken()
}

func (p *Parser) Expect(kind TokenKind) (Token, *Error) {
	if p.CurrentToken.Kind == kind {
		token := p.CurrentToken
		p.Advance()
		return token, nil
	}

	return Token{}, NewError(
		fmt.Errorf("expected '%s' got '%s'", kind.DebugString(), p.CurrentToken.DebugString()),
		p.CurrentToken,
		p.tokenizer.TokenizerData,
		p.tokenizer.SourceCode,
	)
}

func (p *Parser) Statements() []AstNode {

	var statements []AstNode = nil

	for !p.tokenizer.Eof() {
		statement := p.Statement()
		statements = append(statements, statement)
		if _, err := p.Expect(';'); err != nil {
			statements = append(statements, err)
		}
	}

	return statements
}

func (p *Parser) Statement() (result Statement) {
	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_PRAMGA:
		return p.PragmaStatement()
	case TokenKind_Keyword_CREATE:
		return p.CreateStatement()
	case TokenKind_Keyword_BEGIN:
		return p.BeginStatement()
	case TokenKind_Keyword_COMMIT:
		p.Advance()
		return &CommitTransaction{}
	default:
		return NewError(
			fmt.Errorf("expected statement opening keyword 'create', 'drop' or 'alter'"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) BeginStatement() (result Statement) {

	if _, err := p.Expect(TokenKind_Keyword_BEGIN); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_TRANSACTION); err != nil {
		return err
	}

	return &BeginTransaction{}
}

func (p *Parser) PragmaStatement() (result Statement) {

	if _, err := p.Expect(TokenKind_Keyword_PRAMGA); err != nil {
		return err
	}

	pragmaIdentifier := p.CatalogObjectIdentifier()
	if err, isErr := pragmaIdentifier.(*Error); isErr {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case '=':
		{
			p.Advance()
			pragmaValue := p.PragmaValue()
			return &Pragma{
				Name:  pragmaIdentifier,
				Value: pragmaValue,
			}
		}
	case '(':
		{
			p.Advance()
			pragmaValue := p.PragmaValue()
			if _, err := p.Expect(')'); err != nil {
				return err
			}
			return &Pragma{
				Name:  pragmaIdentifier,
				Value: pragmaValue,
			}
		}
	default:
		p.Advance()
		return NewError(
			fmt.Errorf("expected '=' or parentheses for pragma value"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) PragmaValue() (result PragmaValue) {
	switch token := p.CurrentToken; token.Kind {
	case TokenKind_DecimalNumericLiteral:
		p.Advance()
		return &LiteralNumber{
			Token: token,
			Value: p.TokenToNumber(token),
		}
	case TokenKind_Identifier:
		p.Advance()
		return &token
	case TokenKind_StringLiteral:
		p.Advance()
		return &LiteralString{
			Token: token,
			Value: token.Text,
		}
	case TokenKind_Keyword_TRUE, TokenKind_Keyword_FALSE, TokenKind_Keyword_ON:
		p.Advance()
		return &LiteralBoolean{
			Token: token,
			Value: p.TokenToBoolean(token),
		}
	default:
		if p.PeekedToken.Kind == ';' {
			// assume we got a value, to keep the parser going.
			p.Advance()
		}
		return NewError(
			fmt.Errorf("expected pragma value"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) CreateStatement() (result Statement) {

	if _, err := p.Expect(TokenKind_Keyword_CREATE); err != nil {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(false)
	case TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(false)
	case TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(false)
	case TokenKind_Keyword_INDEX:
		return p.CreateIndexStatement(false)
	case TokenKind_Keyword_UNIQUE:
		return p.CreateIndexStatement(true)
	case TokenKind_Keyword_VIRTUAL:
		return p.CreateVirtualTableStatement()
	case TokenKind_Keyword_TEMPORARY:
		return p.CreateTemporaryStatement()
	default:
		panic(fmt.Sprintf("'create '%s' statement is not implemented", token.Text))
	}
}

func (p *Parser) CreateTemporaryStatement() Statement {

	if _, err := p.Expect(TokenKind_Keyword_TEMPORARY); err != nil {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_TABLE:
		return p.CreateTableStatement(true)
	case TokenKind_Keyword_VIEW:
		return p.CreateViewStatement(true)
	case TokenKind_Keyword_TRIGGER:
		return p.CreateTriggerStatement(true)
	default:
		return NewError(
			errors.New("unexpected token after 'temporary' keyword"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) CreateViewStatement(temporary bool) (result Statement) {

	if temporary {
		if _, err := p.Expect(TokenKind_Keyword_TEMPORARY); err != nil {
			return err
		}
	}

	if _, err := p.Expect(TokenKind_Keyword_VIEW); err != nil {
		return err
	}

	_, ifnotexists := p.IfNotExists().(*IfNotExists)

	viewIdentifier := p.CatalogObjectIdentifier()

	columnNames := []Identifier{}
	if p.CurrentToken.Kind == '(' {

	ColumnNamesLoop:
		for !p.tokenizer.Eof() {
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

		if _, err := p.Expect(')'); err != nil {
			return err
		}
	}

	if _, err := p.Expect(TokenKind_Keyword_AS); err != nil {
		return err
	}

	selectStmt := p.SelectStatement()
	return &CreateView{
		IfNotExists:    ifnotexists,
		Columns:        columnNames,
		ViewIdentifier: viewIdentifier,
		AsSelect:       selectStmt,
	}
}

func (p *Parser) CreateTriggerStatement(temporary bool) (result Statement) {

	if _, err := p.Expect(TokenKind_Keyword_TRIGGER); err != nil {
		return err
	}

	for p.CurrentToken.Kind != TokenKind_Keyword_BEGIN {
		p.Advance()
	}
	p.Advance()
	for p.CurrentToken.Kind != ';' {
		p.Advance()
	}
	p.Advance()
	for p.CurrentToken.Kind != TokenKind_Keyword_END {
		p.Advance()
	}
	p.Advance()

	return &CreateTrigger{}
}

func (p *Parser) SelectStatement() Statement {

	if _, err := p.Expect(TokenKind_Keyword_SELECT); err != nil {
		return err
	}

	for !p.tokenizer.Eof() {
		if p.CurrentToken.Kind == ';' {
			break
		}
		p.Advance()
	}

	return &Select{}
}

func (p *Parser) CreateTableStatement(isTemporary bool) Statement {

	if _, err := p.Expect(TokenKind_Keyword_TABLE); err != nil {
		return err
	}

	_, ifnotexists := p.IfNotExists().(*IfNotExists)

	tableIdentifier := p.CatalogObjectIdentifier()
	if err, isErr := tableIdentifier.(*Error); isErr {
		return err
	}

	tableDefinition := p.TableDefinition()
	if err, isErr := tableDefinition.(*Error); isErr {
		return err
	}

	tableOptions := p.TableOptions()

	return &CreateTable{
		IsTemporary:     isTemporary,
		IfNotExist:      ifnotexists,
		TableIdentifier: tableIdentifier,
		TableDefinition: tableDefinition,
		TableOptions:    tableOptions,
	}
}

func (p *Parser) CreateVirtualTableStatement() Statement {

	if _, err := p.Expect(TokenKind_Keyword_VIRTUAL); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_TABLE); err != nil {
		return err
	}

	_, ifnotexists := p.IfNotExists().(*IfNotExists)

	tableIdentifier := p.CatalogObjectIdentifier()
	if err, isErr := tableIdentifier.(*Error); isErr {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_USING); err != nil {
		return err
	}

	moduleName := p.Identifier()
	if err, isErr := moduleName.(*Error); isErr {
		return err
	}

	args := []string{}
	if p.CurrentToken.Kind == '(' {
		p.Advance()
		str := string("")
	ModuleArgsLoop:
		for !p.tokenizer.Eof() {
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
				str = strings.Join([]string{str, token.Text}, "")
			}
		}

		if _, err := p.Expect(')'); err != nil {
			return err
		}
	}

	return &CreateVirtualTable{
		IfNotExist:      ifnotexists,
		TableIdentifier: tableIdentifier,
		ModuleName:      moduleName,
		ModuleArgs:      args,
	}
}

func (p *Parser) CreateIndexStatement(unique bool) Statement {

	if unique {
		p.Expect(TokenKind_Keyword_UNIQUE)
	}

	if _, err := p.Expect(TokenKind_Keyword_INDEX); err != nil {
		return err
	}

	_, ifnotexists := p.IfNotExists().(*IfNotExists)

	indexIdentifier := p.CatalogObjectIdentifier()
	if err, isErr := indexIdentifier.(*Error); isErr {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_ON); err != nil {
		return err
	}

	tableName := p.Identifier()
	if err, isErr := tableName.(*Error); isErr {
		return err
	}

	if _, err := p.Expect('('); err != nil {
		return err
	}

	indexedColumns := []AstNode{}
IndexedColumnsLoop:
	for !p.tokenizer.Eof() {
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

	if _, err := p.Expect(')'); err != nil {
		return err
	}

	var whereExpr Expr = nil
	if p.CurrentToken.Kind == TokenKind_Keyword_WHERE {
		p.Advance()
		whereExpr = p.Expr(0)
	}

	return &CreateIndex{
		Unique:          unique,
		IfNotExists:     ifnotexists,
		IndexIdentifier: indexIdentifier,
		OnTable:         tableName,
		IndexedColumns:  indexedColumns,
		WhereExpr:       whereExpr,
	}
}

func (p *Parser) IndexedColumn(allowExpressions bool) AstNode {

	var indexSubject AstNode = nil
	if allowExpressions {
		indexSubject = p.Expr(0)
	} else {
		indexSubject = p.Identifier()
	}

	var collationName Identifier = nil
	if p.CurrentToken.Kind == TokenKind_Keyword_COLLATE {
		p.Advance()
		collationName := p.Identifier()
		if err, isErr := collationName.(*Error); isErr {
			return err
		}
	}

	order := p.MaybeOrderBy()

	return &IndexedColumn{
		Subject:       indexSubject,
		CollationName: collationName,
		Order:         order,
	}
}

func (p *Parser) IfNotExists() AstNode {
	if p.CurrentToken.Kind != TokenKind_Keyword_IF {
		return nil
	}
	p.Advance()

	if _, err := p.Expect(TokenKind_Keyword_NOT); err != nil {
		return err
	}
	if _, err := p.Expect(TokenKind_Keyword_EXISTS); err != nil {
		return err
	}

	return &IfNotExists{}
}

func (p *Parser) CatalogObjectIdentifier() AstNode {

	schemaOrTable := p.Identifier()
	if err, isErr := schemaOrTable.(*Error); isErr {
		return NewError(
			fmt.Errorf("%w: for 'schema' or 'object (table, index, trigger, view, etc)' name", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	if p.CurrentToken.Kind != TokenKind_Period {
		return &CatalogObjectIdentifier{
			SchemaName: nil,
			ObjectName: schemaOrTable,
		}
	}
	p.Advance()

	schema := schemaOrTable
	table := p.Identifier()
	if err, isErr := table.(*Error); isErr {
		return NewError(
			fmt.Errorf("%w: for 'object (table, index, trigger, view, etc)' name", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	return &CatalogObjectIdentifier{
		SchemaName: schema,
		ObjectName: table,
	}
}

func (p *Parser) TableDefinition() AstNode {
	if _, err := p.Expect('('); err != nil {
		return NewError(
			fmt.Errorf("%w: starting of table definition", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	columnDefs := p.ColumnDefinitions()
	tableConstraints := p.TableConstraints()

	if _, err := p.Expect(')'); err != nil {
		return NewError(
			fmt.Errorf("%w: ending of table definition", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	return &TableDefinition{
		ColumnDefinitions: columnDefs,
		TableConstraints:  tableConstraints,
	}
}

func (p *Parser) ColumnDefinitions() []AstNode {

	definitions := []AstNode{}

ColumnDefinitionsLoop:
	for !p.tokenizer.Eof() {
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

func isConstraintKeyword(token Token) bool {
	_, ok := constaintKeywords[token.Kind]
	return ok
}

func (p *Parser) ColumnDefinition() AstNode {

	columnName := p.Identifier()
	typeName := p.TypeName()
	columnConstraints := p.ColumnConstraints()

	return &ColumnDefinition{
		ColumnName:        columnName,
		TypeName:          typeName,
		ColumnConstraints: columnConstraints,
	}
}

func (p *Parser) ColumnConstraints() []AstNode {

	result := []AstNode{}

ColumnConstraintsLoop:
	for !p.tokenizer.Eof() {
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

func (p *Parser) ColumnConstraint() AstNode {

	constraintName := p.MaybeConstraintName()

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_PRIMARY:
		return p.ColumnConstraint_PrimaryKey(constraintName)
	case TokenKind_Keyword_NOT:
		return p.ColumnConstraint_NotNull(constraintName)
	case TokenKind_Keyword_DEFAULT:
		return p.ColumnConstraint_Default(constraintName)
	case TokenKind_Keyword_UNIQUE:
		return p.ColumnConstraint_Unique(constraintName)
	case TokenKind_Keyword_COLLATE:
		return p.ColumnConstraint_Collate(constraintName)
	case TokenKind_Keyword_CHECK:
		return p.Constraint_Check(constraintName)
	case TokenKind_Keyword_AS:
		return p.ColumnConstraint_Generated(constraintName)
	case TokenKind_Keyword_GENERATED:
		return p.ColumnConstraint_Generated(constraintName)
	default:
		{
			return NewError(
				errors.New("expected column constraint"),
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
	}
}

func (p *Parser) ColumnConstraint_PrimaryKey(constraintName AstNode) ColumnConstraint {

	if _, err := p.Expect(TokenKind_Keyword_PRIMARY); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_KEY); err != nil {
		return err
	}

	orderBy := p.MaybeOrderBy()
	conflictclause := p.MaybeConflictClause()

	autoincrement := false
	if p.CurrentToken.Kind == TokenKind_Keyword_AUTOINCREMENT {
		p.Advance()
	}

	return &ColumnConstraint_PrimaryKey{
		Name:           constraintName,
		ConflictClause: conflictclause,
		Order:          orderBy,
		AutoIncrement:  autoincrement,
	}
}

func (p *Parser) ColumnConstraint_NotNull(constraintName AstNode) ColumnConstraint {

	if _, err := p.Expect(TokenKind_Keyword_NOT); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_NULL); err != nil {
		return err
	}

	return &ColumnConstraint_NotNull{}
}

func (p *Parser) ColumnConstraint_Default(constraintName AstNode) ColumnConstraint {

	if _, err := p.Expect(TokenKind_Keyword_DEFAULT); err != nil {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_StringLiteral:
		p.Advance()
		return &ColumnConstraint_Default{
			Default: &LiteralString{
				Token: token,
				Value: token.Text,
			},
		}
	case TokenKind_DecimalNumericLiteral, TokenKind_BinaryNumericLiteral, TokenKind_HexNumericLiteral, TokenKind_OctalNumericLiteral:
		p.Advance()
		return &ColumnConstraint_Default{
			Default: &LiteralNumber{
				Token: token,
				Value: p.TokenToNumber(token),
			},
		}
	case TokenKind_Keyword_TRUE, TokenKind_Keyword_FALSE:
		p.Advance()
		return &ColumnConstraint_Default{
			Default: &LiteralBoolean{
				Token: token,
				Value: p.TokenToBoolean(token),
			},
		}
	case TokenKind_Identifier:
		p.Advance()
		return &ColumnConstraint_Default{
			Default: &token,
		}
	default:
		panic("not implemented")
	}
}

func (p *Parser) TokenToBoolean(token Token) Boolean {
	switch token.Kind {
	case TokenKind_Keyword_TRUE:
		return Boolean(true)
	case TokenKind_Keyword_FALSE:
		return Boolean(false)
	case TokenKind_Keyword_ON:
		return Boolean(true)
	default:
		panic("unreachable")
	}
}

func (p *Parser) TokenToNumber(token Token) AstNode {
	switch token.Kind {
	case TokenKind_HexNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 16, 64)
		if err != nil {
			return NewError(
				err,
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
		return Integer(value)
	case TokenKind_OctalNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 8, 64)
		if err != nil {
			return NewError(
				err,
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
		return Integer(value)
	case TokenKind_BinaryNumericLiteral:
		value, err := strconv.ParseUint(token.Text, 2, 64)
		if err != nil {
			return NewError(
				err,
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
		return Integer(value)
	case TokenKind_DecimalNumericLiteral:
		value, err := strconv.ParseFloat(token.Text, 10)
		if err != nil {
			return NewError(
				err,
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
		return Float(value)
	default:
		return NewError(
			fmt.Errorf("invalid token kind to parse into number"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) ColumnConstraint_Unique(constraintName AstNode) ColumnConstraint {

	if _, err := p.Expect(TokenKind_Keyword_UNIQUE); err != nil {
		return err
	}

	return &ColumnConstraint_Unique{
		Name: constraintName,
	}
}

func (p *Parser) ColumnConstraint_Collate(constraintName AstNode) ColumnConstraint {

	if _, err := p.Expect(TokenKind_Keyword_COLLATE); err != nil {
		return err
	}

	collationName := p.Identifier()
	if err, isErr := collationName.(*Error); isErr {
		return err
	}
	return &ColumnConstraint_Collate{
		Name:    constraintName,
		Collate: collationName,
	}
}

func (p *Parser) ColumnConstraint_Generated(constraintName AstNode) ColumnConstraint {

	if p.CurrentToken.Kind == TokenKind_Keyword_GENERATED {
		p.Advance()

		if _, err := p.Expect(TokenKind_Keyword_ALWAYS); err != nil {
			return err
		}
	}

	if _, err := p.Expect(TokenKind_Keyword_AS); err != nil {
		return err
	}

	if _, err := p.Expect('('); err != nil {
		return err
	}

	expr := p.Expr(0)

	if _, err := p.Expect(')'); err != nil {
		return err
	}

	storage := p.GeneratedColumnStorage()

	return &ColumnConstraint_Generated{
		Name:    constraintName,
		As:      expr,
		Storage: storage,
	}
}

func (p *Parser) GeneratedColumnStorage() AstNode {
	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_VIRTUAL:
		p.Advance()
		return &GeneratedColumnStorage{
			Token: token,
			Value: GeneratedColumnStorageValue_Virtual,
		}
	case TokenKind_Keyword_STORED:
		p.Advance()
		return &GeneratedColumnStorage{
			Token: token,
			Value: GeneratedColumnStorageValue_Stored,
		}
	default:
		return nil
	}
}

func (p *Parser) MaybeConstraintName() AstNode {
	// we may or may not have a constraint keyword here so peek and check
	if p.CurrentToken.Kind != TokenKind_Keyword_CONSTRAINT {
		return nil
	}
	p.Advance()

	named := p.Identifier()
	if err, isErr := named.(*Error); isErr {
		return NewError(
			fmt.Errorf("%w: for table constraint name", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	return named
}

func (p *Parser) TableConstraints() []AstNode {

	var constraints []AstNode = make([]AstNode, 0)

TableConstraintsLoop:
	for !p.tokenizer.Eof() {

		switch p.CurrentToken.Kind {
		case ')':
			break TableConstraintsLoop
		case ',':
			p.Advance()
			continue
		default:
			tableConstraint := p.TableConstraint()
			constraints = append(constraints, tableConstraint)
		}
	}

	return constraints
}

func (p *Parser) TableConstraint() AstNode {

	constraintName := p.MaybeConstraintName()

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_PRIMARY:
		return p.TableConstraint_PrimaryKey(constraintName)
	case TokenKind_Keyword_FOREIGN:
		return p.TableConstraint_ForeignKey(constraintName)
	default:
		return NewError(
			errors.New("expected table constraint"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) TableConstraint_PrimaryKey(constraintName AstNode) TableConstraint {

	if _, err := p.Expect(TokenKind_Keyword_PRIMARY); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_KEY); err != nil {
		return err
	}

	if _, err := p.Expect('('); err != nil {
		return err
	}

	var indexedCols []AstNode = make([]AstNode, 0)

IndexedColumnsLoop:
	for !p.tokenizer.Eof() {

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

	if _, err := p.Expect(')'); err != nil {
		return err
	}

	conflictclause := p.MaybeConflictClause()

	tableConstraint := &TableConstraint_PrimaryKey{
		Name:           constraintName,
		IndexedColumns: indexedCols,
		ConflictClause: conflictclause,
	}

	return tableConstraint
}

func (p *Parser) TableConstraint_ForeignKey(constraintName AstNode) TableConstraint {

	if _, err := p.Expect(TokenKind_Keyword_FOREIGN); err != nil {
		return err
	}

	if _, err := p.Expect(TokenKind_Keyword_KEY); err != nil {
		return err
	}

	if _, err := p.Expect('('); err != nil {
		return err
	}

	columnNames := []AstNode{}

ColumnNamesLoop:
	for !p.tokenizer.Eof() {
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

	if _, err := p.Expect(')'); err != nil {
		return err
	}

	fkClause := p.ForeignKeyClause()

	return &TableConstraint_ForeignKey{
		Name:     constraintName,
		Columns:  columnNames,
		FkClause: fkClause,
	}
}

func (p *Parser) ForeignKeyClause() AstNode {
	if _, err := p.Expect(TokenKind_Keyword_REFERENCES); err != nil {
		return err
	}

	foreignTable := p.Identifier()
	if err, isErr := foreignTable.(*Error); isErr {
		return err
	}

	foreignColumns := []AstNode{}

	if p.CurrentToken.Kind == '(' {
		p.Advance()
	ForeignColumnsLoop:
		for !p.tokenizer.Eof() {
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

		if _, err := p.Expect(')'); err != nil {
			return err
		}
	}

	actions := []ForeignKeyActionTrigger{}
	var matchName AstNode = nil
	var deferrable AstNode = nil

ForeignKeyModifiersLoop:
	for !p.tokenizer.Eof() {
		switch p.CurrentToken.Kind {
		case TokenKind_Keyword_ON:
			action := p.ForeignKeyActionTrigger()
			actions = append(actions, action)
			continue
		case TokenKind_Keyword_MATCH:
			matchName = p.Identifier()
			continue
		case TokenKind_Keyword_NOT:
			deferrable = p.ForeignKeyDeferrable()
			continue
		case TokenKind_Keyword_DEFERRABLE:
			deferrable = p.ForeignKeyDeferrable()
			continue
		default:
			break ForeignKeyModifiersLoop
		}
	}

	return &ForeignKeyClause{
		ForeignTable:   foreignTable,
		ForeignColumns: foreignColumns,
		Actions:        actions,
		MatchName:      matchName,
		Deferrable:     deferrable,
	}
}

func (p *Parser) ForeignKeyActionTrigger() ForeignKeyActionTrigger {

	if _, err := p.Expect(TokenKind_Keyword_ON); err != nil {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_DELETE:
		p.Advance()
		return &OnDelete{
			Action: p.ForeignKeyAction(),
		}
	case TokenKind_Keyword_UPDATE:
		p.Advance()
		return &OnUpdate{
			Action: p.ForeignKeyAction(),
		}
	default:
		return NewError(
			fmt.Errorf("expected action trigger keyword 'delete' or 'update' for fk action"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) ForeignKeyAction() ForeignKeyAction {
	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_CASCADE:
		p.Advance()
		return &Cascade{}
	case TokenKind_Keyword_RESTRICT:
		p.Advance()
		return &Restrict{}
	case TokenKind_Keyword_NO:
		p.Advance()
		if _, err := p.Expect(TokenKind_Keyword_ACTION); err != nil {
			return err
		}
		return &NoAction{}
	case TokenKind_Keyword_SET:
		p.Advance()
		switch token := p.CurrentToken; token.Kind {
		case TokenKind_Keyword_DEFAULT:
			p.Advance()
			return &SetDefault{}
		case TokenKind_Keyword_NULL:
			p.Advance()
			return &SetNull{}
		default:
			return NewError(
				fmt.Errorf("expected keyword 'default' or 'null' for fk action 'set'"),
				token,
				p.tokenizer.TokenizerData,
				p.tokenizer.SourceCode,
			)
		}
	default:
		return NewError(
			fmt.Errorf("expected fk action method 'cascade', 'restrict', 'no action', 'set default' or 'set null'"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) ForeignKeyDeferrable() AstNode {

	not := p.CurrentToken.Kind == TokenKind_Keyword_NOT
	if not {
		p.Advance()
	}

	if _, err := p.Expect(TokenKind_Keyword_DEFERRABLE); err != nil {
		return err
	}

	if p.CurrentToken.Kind == TokenKind_Keyword_INITIALLY {
		p.Advance()
		switch token := p.CurrentToken; token.Kind {
		case TokenKind_Keyword_IMMEDIATE:
			p.Advance()
			return ForeignKeyDeferrable_Immediate
		case TokenKind_Keyword_DEFERRED:
			p.Advance()
			if not {
				return ForeignKeyDeferrable_Immediate
			} else {
				return ForeignKeyDeferrable_Deferred
			}
		}
	}

	return ForeignKeyDeferrable_Immediate
}

func (p *Parser) TableOptions() AstNode {

	strict := false
	withoutRowId := false

TableOptionsLoop:
	for !p.tokenizer.Eof() {
		switch token := p.CurrentToken; token.Kind {
		case TokenKind_Keyword_STRICT:
			p.Advance()
			strict = true
			continue
		case TokenKind_Keyword_WITHOUT:
			p.Advance()
			if _, err := p.Expect(TokenKind_Keyword_ROWID); err != nil {
				return err
			}
		default:
			break TableOptionsLoop
		}
	}

	return &TableOptions{
		Strict:       strict,
		WithoutRowId: withoutRowId,
	}
}

func (p *Parser) MaybeConflictClause() AstNode {

	if p.CurrentToken.Kind != TokenKind_Keyword_ON {
		return nil
	}
	p.Advance()

	if _, err := p.Expect(TokenKind_Keyword_CONFLICT); err != nil {
		return err
	}

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_ROLLBACK:
		fallthrough
	case TokenKind_Keyword_ABORT:
		fallthrough
	case TokenKind_Keyword_FAIL:
		fallthrough
	case TokenKind_Keyword_IGNORE:
		fallthrough
	case TokenKind_Keyword_REPLACE:
		p.Advance()
		return &token
	default:
		return NewError(
			errors.New("expected conflict clause verb"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) TypeName() AstNode {

	ident := p.Identifier()
	if err, isErr := ident.(*Error); isErr {
		return NewError(
			fmt.Errorf("%w: for type name", err),
			err.OffendingToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	return &TypeName{
		TypeName: ident,
	}
}

func (p *Parser) Constraint_Check(constraintName AstNode) Constraint {

	if _, err := p.Expect(TokenKind_Keyword_CHECK); err != nil {
		return err
	}

	if _, err := p.Expect('('); err != nil {
		return err
	}

	expr := p.Expr(0)
	if err, isErr := expr.(*Error); isErr {
		return &Constraint_Check{
			Name:  constraintName,
			Check: err,
		}
	}

	if _, err := p.Expect(')'); err != nil {
		return NewError(
			fmt.Errorf("%w: in check constraint", err),
			p.CurrentToken,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}

	return &Constraint_Check{
		Name:  constraintName,
		Check: expr,
	}
}

func (p *Parser) MaybeOrderBy() OrderBy {

	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Keyword_ASC:
		fallthrough
	case TokenKind_Keyword_DESC:
		p.Advance()
		return (OrderBy)(&token)
	default:
		return (OrderBy)(nil)
	}
}

func (p *Parser) Identifier() Identifier {
	if token := p.CurrentToken; token.Kind == TokenKind_Identifier {
		p.Advance()
		return &token
	}

	return NewError(
		errors.New("expected identifier"),
		p.CurrentToken,
		p.tokenizer.TokenizerData,
		p.tokenizer.SourceCode,
	)
}

func (p *Parser) MaybeBinaryOperator() BinaryOperator {
	switch p.CurrentToken.Kind {
	case '=':
		return &EquivOp{}
	case '+':
		return &AddOp{}
	case '-':
		return &SubOp{}
	case '*':
		return &MulOp{}
	case '/':
		return &DivOp{}
	case TokenKind_gte:
		return &GteOp{}
	case TokenKind_Keyword_IN:
		return &InOp{}
	default:
		return nil
	}
}

func (p *Parser) Expr(minBindingPower int) Expr {

	lhs := p.Nud()

	for !p.tokenizer.Eof() {
		operator := p.MaybeBinaryOperator()
		if operator == nil {
			return lhs
		}
		if operator.bindingPower().L < minBindingPower {
			break
		}

		// consume the operator found in MaybeBinaryOperator
		p.Advance()

		rhs := p.Expr(operator.bindingPower().R)
		operator.SetLhs(lhs)
		operator.SetRhs(rhs)
		lhs = operator
	}

	return lhs
}

func (p *Parser) Identifier_Nud() Expr {
	ident := p.CurrentToken
	p.Advance()

	switch p.CurrentToken.Kind {
	case '(':
		p.Advance()
		arguments := p.ExprList()
		if _, err := p.Expect(')'); err != nil {
			return err
		}
		return &FunctionCall{
			Name: ident,
			Args: arguments,
		}
	case '.':
		p.Advance()
		schemaOrTable := ident
		tableOrColumn := p.Identifier()
		if err, isErr := tableOrColumn.(*Error); isErr {
			return err
		}
		if p.CurrentToken.Kind == '.' {
			p.Advance()
			return &ColumnName{
				Schema: (Identifier)(nil),
				Table:  &schemaOrTable,
				Column: tableOrColumn,
			}
		}

		column := p.Identifier()
		if err, isErr := column.(*Error); isErr {
			return err
		}

		return &ColumnName{
			Schema: &schemaOrTable,
			Table:  tableOrColumn,
			Column: column,
		}
	default:
		return &ident
	}
}

func (p *Parser) Nud() Expr {
	switch token := p.CurrentToken; token.Kind {
	case TokenKind_Identifier:
		{
			return p.Identifier_Nud()
		}
	case TokenKind_DecimalNumericLiteral, TokenKind_BinaryNumericLiteral, TokenKind_OctalNumericLiteral, TokenKind_HexNumericLiteral:
		p.Advance()
		return &LiteralNumber{
			Token: token,
			Value: p.TokenToNumber(token),
		}
	case TokenKind_StringLiteral:
		p.Advance()
		return &LiteralString{
			Token: token,
			Value: token.Text,
		}
	case TokenKind_Keyword_TRUE, TokenKind_Keyword_FALSE:
		p.Advance()
		return &LiteralBoolean{
			Token: token,
			Value: p.TokenToBoolean(token),
		}
	case TokenKind_Keyword_CASE:
		return p.CaseExpr()
	case '(':
		p.Advance()
		result := p.Expr(0)

		if p.CurrentToken.Kind == ',' {
			p.Advance()
			list := ExprList{result}

			rest := p.ExprList()
			result = append(list, rest...)
		}
		if _, err := p.Expect(')'); err != nil {
			return err
		}
		return result
	default:
		return NewError(
			fmt.Errorf("expected leaf for left handside of expression"),
			token,
			p.tokenizer.TokenizerData,
			p.tokenizer.SourceCode,
		)
	}
}

func (p *Parser) CaseExpr() Expr {

	if _, err := p.Expect(TokenKind_Keyword_CASE); err != nil {
		return err
	}

	var operand Expr = nil
	if p.CurrentToken.Kind != TokenKind_Keyword_WHEN {
		operand = p.Expr(0)
	}

	cases := []WhenThen{}

CasesLoop:
	for !p.tokenizer.Eof() {
		if _, err := p.Expect(TokenKind_Keyword_WHEN); err != nil {
			return err
		}
		when := p.Expr(0)

		if _, err := p.Expect(TokenKind_Keyword_THEN); err != nil {
			return err
		}
		then := p.Expr(0)

		cases = append(cases, WhenThen{
			When: when,
			Then: then,
		})

		switch p.CurrentToken.Kind {
		case TokenKind_Keyword_ELSE, TokenKind_Keyword_END:
			break CasesLoop
		}
	}

	var elseExpr Expr = nil
	if p.CurrentToken.Kind == TokenKind_Keyword_ELSE {
		p.Advance()
		elseExpr = p.Expr(0)
	}

	if _, err := p.Expect(TokenKind_Keyword_END); err != nil {
		return err
	}

	return &CaseExpression{
		Operand: operand,
		Cases:   cases,
		Else:    elseExpr,
	}
}

func (p *Parser) ExprList() ExprList {
	result := ExprList{}

ExprListLoop:
	for !p.tokenizer.Eof() {
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
