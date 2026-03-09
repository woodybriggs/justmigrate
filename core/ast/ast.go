package ast

import (
	"errors"
	"strconv"
	"woodybriggs/justmigrate/core/tik"
)

type Identifier tik.Token

func MakeIdentifier(token tik.Token) *Identifier {
	ident := Identifier(token)
	return &ident
}

func (t *Identifier) node()            {}
func (t *Identifier) nodeExpression()  {}
func (t *Identifier) nodeIdentifier()  {}
func (t *Identifier) nodePragmaValue() {}

func (t Identifier) ToStringLiteral() LiteralString {
	return LiteralString{
		Token: tik.Token(t),
		Value: t.Text,
	}
}

func (node *Identifier) Eq(other Expr) bool {
	if other == nil {
		return false
	}

	switch t := other.(type) {
	case *Identifier:
		return node.Text == t.Text
	case Expr:
		return node.AsExpr().Eq(t)
	default:
		return false
	}
}

func (node *Identifier) AsExpr() Expr {
	return Expr(node)
}

type Keyword tik.Token

func MakeKeyword(token tik.Token) *Keyword {
	result := Keyword(token)
	return &result
}

type IfExists struct {
	If     Keyword
	Exists Keyword
}

type DropTable struct {
	IfExists        *IfExists
	TableIdentifier CatalogObjectIdentifier
}

type AlterTable struct {
	AlterKeyword    Keyword
	TableKeyword    Keyword
	TableIdentifier *CatalogObjectIdentifier
	Alteration      TableAlteration
}

type AddColumn struct {
	AddKeyword       Keyword
	ColumnKeyword    *Keyword
	ColumnDefinition ColumnDefinition
}

type DropColumn struct {
	DropKeyword   Keyword
	ColumnKeyword *Keyword
	ColumnName    Identifier
}

type Pragma struct {
	Name  CatalogObjectIdentifier
	Value Expr
}

type BeginTransaction struct{}

type CommitTransaction struct{}

type Select struct{}

type CreateTable struct {
	CreateKeyword   Keyword
	TableKeyword    Keyword
	Temporary       *Keyword
	IfNotExist      *IfNotExists
	TableIdentifier *CatalogObjectIdentifier
	TableDefinition *TableDefinition
	TableOptions    *TableOptions
}

func MakeCreateTable(
	create Keyword,
	temporary *Keyword,
	table Keyword,
	ifNotExists *IfNotExists,
	tableIdent *CatalogObjectIdentifier,
	tableDefinition *TableDefinition,
	tableOptions *TableOptions,
) *CreateTable {
	return &CreateTable{
		CreateKeyword:   create,
		Temporary:       temporary,
		TableKeyword:    table,
		IfNotExist:      ifNotExists,
		TableIdentifier: tableIdent,
		TableDefinition: tableDefinition,
		TableOptions:    tableOptions,
	}
}

type CreateVirtualTable struct {
	IfNotExist      *IfNotExists
	TableIdentifier CatalogObjectIdentifier
	ModuleName      Identifier
	ModuleArgs      []string
}

func (node *CreateVirtualTable) nodeStatement() {}

type CreateIndex struct {
	CreateKeyword   Keyword
	IndexKeyword    Keyword
	UniqueKeyword   *Keyword
	IfNotExists     *IfNotExists
	IndexIdentifier CatalogObjectIdentifier
	OnTable         Identifier
	IndexedColumns  []IndexedColumn
	WhereExpr       Expr
}

func MakeCreateIndex(
	createKeyword Keyword,
	uniqueKeyword *Keyword,
	indexKeyword Keyword,
	ifNotExists *IfNotExists,
	indexIdentifier *CatalogObjectIdentifier,
	onKeyword Keyword,
	tableName Identifier,
	lParen tik.Token,
	indexedCols []IndexedColumn,
	rParen tik.Token,
	whereKeyword *Keyword,
	whereExpr Expr,
) *CreateIndex {
	return &CreateIndex{
		CreateKeyword:   createKeyword,
		UniqueKeyword:   uniqueKeyword,
		IfNotExists:     ifNotExists,
		IndexIdentifier: *indexIdentifier,
		OnTable:         tableName,
		IndexedColumns:  indexedCols,
		WhereExpr:       whereExpr,
	}
}

func (node *CreateIndex) nodeStatement() {}

type TriggerTime interface {
	triggerTime()
}

type TriggerEvent interface {
	triggerEvent()
}

type CreateTrigger struct {
	CreateKeyword     Keyword
	TriggerKeyword    Keyword
	Temporary         *Keyword
	IfNotExists       *IfNotExists
	TriggerIdentifier CatalogObjectIdentifier
	TriggerTime       TriggerTime
	TriggerEvent      TriggerEvent
	OnTable           CatalogObjectIdentifier
}

type TriggerTimeBefore struct {
	BeforeKeyword Keyword
}

func (node *TriggerTimeBefore) triggerTime() {}

type TriggerTimeAfter struct {
	AfterKeyword Keyword
}

func (node *TriggerTimeAfter) triggerTime() {}

type TriggerTimeInsteadOf struct {
	InsteadKeyword Keyword
	Of             Keyword
}

func (node *TriggerTimeInsteadOf) triggerTime() {}

type TriggerEventDelete struct {
	DeleteKeyword Keyword
}

func (node *TriggerEventDelete) triggerEvent() {}

type TriggerEventInsert struct {
	InsertKeyword Keyword
}

func (node *TriggerEventInsert) triggerEvent() {}

type TriggerEventUpdate struct {
	UpdateKeyword Keyword
}

func (node *TriggerEventUpdate) triggerEvent() {}

type TriggerEventUpdateOf struct {
	UpdateKeyword Keyword
	Of            Keyword
	Columns       []Identifier
}

func (node *TriggerEventUpdateOf) triggerEvent() {}

type IndexedColumn struct {
	Subject   Expr
	Collation *Collation
	Order     *Keyword
}

type CreateView struct {
	IfNotExists    *IfNotExists
	ViewIdentifier CatalogObjectIdentifier
	Columns        []Identifier
	AsSelect       Statement
}

func (node *CreateView) nodeStatement() {}

type IfNotExists struct {
	If     Keyword
	Not    Keyword
	Exists Keyword
}

func MakeIfNotExists(
	ifKeyword Keyword,
	notKeyword Keyword,
	existsKeyword Keyword,
) *IfNotExists {
	return &IfNotExists{
		If:     ifKeyword,
		Not:    notKeyword,
		Exists: existsKeyword,
	}
}

type CatalogObjectIdentifier struct {
	SchemaName *Identifier
	ObjectName Identifier
}

func MakeCatalogObjectIdentifier(
	schemaName *Identifier,
	objectName Identifier,
) *CatalogObjectIdentifier {
	return &CatalogObjectIdentifier{
		SchemaName: schemaName,
		ObjectName: objectName,
	}
}

// @note(woody): this should actually be a tagged enum
// TableDefinitionAsSelect
// TableDefinitionColumnsConstraints
type TableDefinition struct {
	LParen tik.Token

	// AsSelect *SelectStatement
	ColumnDefinitions []ColumnDefinition
	TableConstraints  []TableConstraint

	RParent tik.Token
}

func MakeTableDefinition(
	lParen tik.Token,
	columnDefs []ColumnDefinition,
	constraints []TableConstraint,
	rParen tik.Token,
) *TableDefinition {
	return &TableDefinition{
		LParen:            lParen,
		ColumnDefinitions: columnDefs,
		TableConstraints:  constraints,
		RParent:           rParen,
	}
}

type TableOptions struct {
	Strict       *Keyword
	WithoutRowId *WithoutRowId
}

func MakeTableOptions(strict *Keyword, withoutRowId *WithoutRowId) *TableOptions {
	return &TableOptions{
		Strict:       strict,
		WithoutRowId: withoutRowId,
	}
}

type WithoutRowId struct {
	Without Keyword
	RowId   Keyword
}

func MakeWithoutRowId(without Keyword, rowId Keyword) *WithoutRowId {
	return &WithoutRowId{
		Without: without,
		RowId:   rowId,
	}
}

type ColumnDefinition struct {
	ColumnName        Identifier
	TypeName          *TypeName
	ColumnConstraints []ColumnConstraint
}

func MakeColumnDefinition(
	name Identifier,
	typ *TypeName,
	constraints []ColumnConstraint,
) *ColumnDefinition {
	return &ColumnDefinition{
		ColumnName:        name,
		TypeName:          typ,
		ColumnConstraints: constraints,
	}
}

type TypeName struct {
	Name Identifier
	Arg0 NumericLiteral
	Arg1 NumericLiteral
}

func MakeTypeName(
	name Identifier,
	arg0 NumericLiteral,
	arg1 NumericLiteral,
) *TypeName {
	return &TypeName{
		Name: name,
		Arg0: arg0,
		Arg1: arg1,
	}
}

type ConflictClause struct {
	OnKeyword       Keyword
	ConflictKeyword Keyword
	Action          Keyword
}

func MakeConflictClause(
	onKeyword Keyword,
	conflictKeyword Keyword,
	actionKeyword Keyword,
) *ConflictClause {
	return &ConflictClause{
		OnKeyword:       onKeyword,
		ConflictKeyword: conflictKeyword,
		Action:          actionKeyword,
	}
}

type TableConstraint_Check struct {
	Name         *ConstraintName
	CheckKeyword Keyword
	LParen       tik.Token
	Expr         Expr
	RParen       tik.Token
}

func MakeTableConstraintCheck(
	constraintName *ConstraintName,
	checkKeyword Keyword,
	lParen tik.Token,
	expr Expr,
	rParen tik.Token,
) *TableConstraint_Check {
	return &TableConstraint_Check{
		Name:         constraintName,
		CheckKeyword: checkKeyword,
		LParen:       lParen,
		Expr:         expr,
		RParen:       rParen,
	}
}

type TableConstraint_PrimaryKey struct {
	Name           *ConstraintName
	PrimaryKeyword Keyword
	KeyKeyword     Keyword
	LParen         tik.Token
	IndexedColumns []IndexedColumn
	AutoIncrement  *Keyword
	RParen         tik.Token
	ConflictClause *ConflictClause
}

func MakeTableConstraintPrimaryKey(
	constraintName *ConstraintName,
	primaryKeyword Keyword,
	keyKeyword Keyword,
	lParen tik.Token,
	indexedColumns []IndexedColumn,
	rParen tik.Token,
	conflictClause *ConflictClause,
	// sqlite
	autoincrement *Keyword,

) *TableConstraint_PrimaryKey {
	return &TableConstraint_PrimaryKey{
		Name:           constraintName,
		PrimaryKeyword: primaryKeyword,
		KeyKeyword:     keyKeyword,
		LParen:         lParen,
		IndexedColumns: indexedColumns,
		AutoIncrement:  autoincrement,
		RParen:         rParen,
		ConflictClause: conflictClause,
	}
}

type TableConstraint_ForeignKey struct {
	Name           *ConstraintName
	ForeignKeyword Keyword
	KeyKeyword     Keyword
	LParen         tik.Token
	Columns        []Identifier
	RParen         tik.Token
	FkClause       ForeignKeyClause
}

func MakeTableConstraintForeignKey(
	constraintName *ConstraintName,
	foreignKeyword Keyword,
	keyKeyword Keyword,
	lParen tik.Token,
	columns []Identifier,
	rParen tik.Token,
	fkClause *ForeignKeyClause,
) *TableConstraint_ForeignKey {
	return &TableConstraint_ForeignKey{
		Name:           constraintName,
		ForeignKeyword: foreignKeyword,
		KeyKeyword:     keyKeyword,
		LParen:         lParen,
		Columns:        columns,
		RParen:         rParen,
		FkClause:       *fkClause,
	}
}

type IdentifierPair struct {
	A, B Identifier
}

func (node *TableConstraint_ForeignKey) pairColumns() []IdentifierPair {

	pairs := make([]IdentifierPair, 0, len(node.Columns))

	for i, localCol := range node.Columns {
		foreignCol := node.FkClause.ForeignColumns[i]
		pairs = append(pairs, IdentifierPair{
			A: localCol,
			B: foreignCol,
		})
	}

	return pairs
}

type ForeignKeyClause struct {
	ReferencesKeyword Keyword
	ForeignTable      CatalogObjectIdentifier
	LParen            tik.Token
	ForeignColumns    []Identifier
	RParen            tik.Token
	Actions           []ForeignKeyAction
	MatchName         *Identifier
	Deferrable        *ForeignKeyDeferrable
}

func MakeForeignKeyClause(
	referencesKeyword Keyword,
	foreignTable CatalogObjectIdentifier,
	lParen tik.Token,
	foreignColumns []Identifier,
	rParen tik.Token,
	actions []ForeignKeyAction,
	matchName *Identifier,
	deferrable *ForeignKeyDeferrable,
) *ForeignKeyClause {
	return &ForeignKeyClause{
		ReferencesKeyword: referencesKeyword,
		ForeignTable:      foreignTable,
		LParen:            lParen,
		ForeignColumns:    foreignColumns,
		RParen:            rParen,
		Actions:           actions,
		MatchName:         matchName,
		Deferrable:        deferrable,
	}
}

type ForeignKeyDeferrable struct {
	NotKeyword        *Keyword
	DeferrableKeyword Keyword
	InitiallyKeyword  *Keyword
	Deferrable        *Keyword
}

func MakeForeignKeyDeferrable(notKeyword *Keyword, deferrableKeyword Keyword, initiallyKeyword *Keyword, value *Keyword) *ForeignKeyDeferrable {
	return &ForeignKeyDeferrable{
		NotKeyword:        notKeyword,
		DeferrableKeyword: deferrableKeyword,
		InitiallyKeyword:  initiallyKeyword,
		Deferrable:        value,
	}
}

type ForeignKeyDeleteAction struct {
	OnKeyword     Keyword
	DeleteKeyword Keyword
	Action        ForeignKeyActionDo
}

func MakeForeignKeyDeleteAction(
	onKeyword Keyword,
	deleteKeyword Keyword,
	do ForeignKeyActionDo,
) *ForeignKeyDeleteAction {
	return &ForeignKeyDeleteAction{
		OnKeyword:     onKeyword,
		DeleteKeyword: deleteKeyword,
		Action:        do,
	}
}

type ForeignKeyUpdateAction struct {
	OnKeyword     Keyword
	UpdateKeyword Keyword
	Action        ForeignKeyActionDo
}

func MakeForeignKeyUpdateAction(
	onKeyword Keyword,
	updateKeyword Keyword,
	do ForeignKeyActionDo,
) *ForeignKeyUpdateAction {
	return &ForeignKeyUpdateAction{
		OnKeyword:     onKeyword,
		UpdateKeyword: updateKeyword,
		Action:        do,
	}
}

type NoAction struct {
	NoKeyword     Keyword
	ActionKeyword Keyword
}

func MakeForeignKeyActionNoAction(noKeyword Keyword, actionKeyword Keyword) *NoAction {
	return &NoAction{
		NoKeyword:     noKeyword,
		ActionKeyword: actionKeyword,
	}
}

type Restrict Keyword

func MakeForeignKeyActionRestrict(restrictKeyword Keyword) *Restrict {
	val := Restrict(restrictKeyword)
	return &val
}

type SetNull struct {
	SetKeyword  Keyword
	NullKeyword Keyword
}

func MakeForeignKeyActionSetNull(setKeyword Keyword, nullKeyword Keyword) *SetNull {
	return &SetNull{
		SetKeyword:  setKeyword,
		NullKeyword: nullKeyword,
	}
}

type SetDefault struct {
	SetKeyword     Keyword
	DefaultKeyword Keyword
}

func MakeForeignKeyActionSetDefault(setKeyword Keyword, defaultKeyword Keyword) *SetDefault {
	return &SetDefault{
		SetKeyword:     setKeyword,
		DefaultKeyword: defaultKeyword,
	}
}

type Cascade Keyword

func MakeForeignKeyActionCascade(cascadeKeyword Keyword) *Cascade {
	val := Cascade(cascadeKeyword)
	return &val
}

type ConstraintName struct {
	ConstraintKeyword Keyword
	Name              Identifier
}

type ColumnConstraint_PrimaryKey struct {
	Name           *ConstraintName
	PrimaryKeyword Keyword
	KeyKeyword     Keyword
	Order          *Keyword
	ConflictClause *ConflictClause

	// sqlite autoincrement
	AutoIncrement *Keyword
}

func MakeColumnConstraintPrimaryKey(
	constraintName *ConstraintName,
	primaryKeyword Keyword,
	keyKeyword Keyword,
	order *Keyword,
	conflictClause *ConflictClause,
	autoIncrement *Keyword,
) *ColumnConstraint_PrimaryKey {
	return &ColumnConstraint_PrimaryKey{
		Name:           constraintName,
		PrimaryKeyword: primaryKeyword,
		KeyKeyword:     keyKeyword,
		Order:          order,
		ConflictClause: conflictClause,
		AutoIncrement:  autoIncrement,
	}
}

type ColumnConstraint_ForeignKey struct {
	Name     *ConstraintName
	FkClause ForeignKeyClause
}

func MakeColumnConstraintForeignKey(
	constraintName *ConstraintName,
	fkClause ForeignKeyClause,
) *ColumnConstraint_ForeignKey {
	return &ColumnConstraint_ForeignKey{
		Name:     constraintName,
		FkClause: fkClause,
	}
}

type ColumnConstraint_Unique struct {
	Name          *ConstraintName
	UniqueKeyword Keyword
}

func MakeColumnConstraintUnique(
	constraintName *ConstraintName,
	uniqueKeyword Keyword,
) *ColumnConstraint_Unique {
	return &ColumnConstraint_Unique{
		Name:          constraintName,
		UniqueKeyword: uniqueKeyword,
	}
}

type ColumnConstraint_Collate struct {
	Name           *ConstraintName
	CollateKeyword Keyword
	CollationName  Identifier
}

func MakeColumnConstraintCollate(
	constraintName *ConstraintName,
	collateKeyword Keyword,
	collationName Identifier,
) *ColumnConstraint_Collate {
	return &ColumnConstraint_Collate{
		Name:           constraintName,
		CollateKeyword: collateKeyword,
		CollationName:  collationName,
	}
}

type ColumnConstraint_NotNull struct {
	Name           *ConstraintName
	NotKeyword     Keyword
	NullKeyword    Keyword
	ConflictClause *ConflictClause
}

func MakeColumnConstraintNotNull(
	constraintName *ConstraintName,
	notKeyword Keyword,
	nullKeyword Keyword,
	conflictClause *ConflictClause,
) *ColumnConstraint_NotNull {
	return &ColumnConstraint_NotNull{
		Name:           constraintName,
		NotKeyword:     notKeyword,
		NullKeyword:    nullKeyword,
		ConflictClause: conflictClause,
	}
}

type ColumnConstraint_Default struct {
	Name           *ConstraintName
	DefaultKeyword Keyword
	Default        Expr
}

func MakeColumnConstraintDefault(
	constraintName *ConstraintName,
	defaultKeyword Keyword,
	expr Expr,
) *ColumnConstraint_Default {
	return &ColumnConstraint_Default{
		Name:           constraintName,
		DefaultKeyword: defaultKeyword,
		Default:        expr,
	}
}

type ColumnConstraint_Generated struct {
	Name             *ConstraintName
	GeneratedKeyword *Keyword
	AlwaysKeyword    *Keyword
	AsKeyword        Keyword
	AsExpr           Expr
	Storage          any
}

func MakeColumnConstraintGenerated(
	constraintName *ConstraintName,
	generatedKeyword *Keyword,
	alwaysKeyword *Keyword,
	asKeyword Keyword,
	asExpr Expr,
	storage any,
) *ColumnConstraint_Generated {
	return &ColumnConstraint_Generated{
		Name:             constraintName,
		GeneratedKeyword: generatedKeyword,
		AlwaysKeyword:    alwaysKeyword,
		AsKeyword:        asKeyword,
		AsExpr:           asExpr,
		Storage:          storage,
	}
}

type ColumnConstraint_Check struct {
	Name         *ConstraintName
	CheckKeyword Keyword
	CheckExpr    Expr
}

func MakeColumnConstraintCheck(
	constraintName *ConstraintName,
	checkKeyword Keyword,
	checkExpr Expr,
) *ColumnConstraint_Check {
	return &ColumnConstraint_Check{
		Name:         constraintName,
		CheckKeyword: checkKeyword,
		CheckExpr:    checkExpr,
	}
}

type ExprList []Expr

func TokenToLiteral(token tik.Token) (Expr, error) {
	switch token.Kind {
	case tik.TokenKind_Keyword_NULL:
		{
			return MakeLiteralNull(token), nil
		}
	case tik.TokenKind_StringLiteral:
		{
			return MakeLiteralString(token, token.Text), nil
		}
	// allow literals to become string literals where a literal is needed
	case tik.TokenKind_Identifier:
		{
			return MakeLiteralString(token, token.Text), nil
		}
	case tik.TokenKind_Keyword_TRUE:
		{
			return MakeLiteralBoolean(token, true), nil
		}
	case tik.TokenKind_Keyword_FALSE:
		{
			return MakeLiteralBoolean(token, false), nil
		}
	case tik.TokenKind_FloatNumericLiteral:
		{
			val, err := strconv.ParseFloat(token.Text, 64)
			if err != nil {
				return nil, err
			}
			return MakeLiteralFloat(token, val), nil
		}
	case tik.TokenKind_IntegerNumericLiteral:
		{
			ival, err := strconv.ParseInt(token.Text, 10, 64)
			if err != nil {
				if errors.Is(err, strconv.ErrRange) {
					uval, err := strconv.ParseUint(token.Text, 10, 64)
					if err != nil {
						return nil, err
					}
					return MakeLiteralUnsignedInteger(token, uval), nil
				}
				return nil, err
			}
			return MakeLiteralSignedInteger(token, ival), nil
		}
	case tik.TokenKind_BinaryNumericLiteral:
		{
			uval, err := strconv.ParseUint(token.Text, 2, 64)
			if err != nil {
				return nil, err
			}
			return MakeLiteralUnsignedInteger(token, uval), nil
		}
	case tik.TokenKind_OctalNumericLiteral:
		{
			uval, err := strconv.ParseUint(token.Text, 8, 64)
			if err != nil {
				return nil, err
			}
			return MakeLiteralUnsignedInteger(token, uval), nil
		}
	case tik.TokenKind_HexNumericLiteral:
		{
			uval, err := strconv.ParseUint(token.Text, 16, 64)
			if err != nil {
				return nil, err
			}
			return MakeLiteralUnsignedInteger(token, uval), nil
		}
	default:
		return nil, errors.New("token can not be converted to literal")
	}
}

type LiteralNull struct {
	Token tik.Token
}

func MakeLiteralNull(token tik.Token) *LiteralNull {
	return &LiteralNull{
		Token: token,
	}
}

type LiteralBoolean struct {
	Token tik.Token
	Value bool
}

func MakeLiteralBoolean(token tik.Token, value bool) *LiteralBoolean {
	return &LiteralBoolean{
		Token: token,
		Value: value,
	}
}

type LiteralUnsignedInteger struct {
	Token tik.Token
	Value uint64
}

func MakeLiteralUnsignedInteger(token tik.Token, value uint64) *LiteralUnsignedInteger {
	return &LiteralUnsignedInteger{
		Token: token,
		Value: value,
	}
}

type LiteralSignedInteger struct {
	Token tik.Token
	Value int64
}

func MakeLiteralSignedInteger(token tik.Token, value int64) *LiteralSignedInteger {
	return &LiteralSignedInteger{
		Token: token,
		Value: value,
	}
}

type LiteralFloat struct {
	Token tik.Token
	Value float64
}

func MakeLiteralFloat(token tik.Token, value float64) *LiteralFloat {
	return &LiteralFloat{
		Token: token,
		Value: value,
	}
}

type LiteralString struct {
	Token tik.Token
	Value string
}

func MakeLiteralString(token tik.Token, value string) *LiteralString {
	return &LiteralString{
		Token: token,
		Value: value,
	}
}

type UnaryOp struct {
	Operator tik.Token
	Rhs      Expr
}

type FunctionCall struct {
	Name Identifier
	Args ExprList
}

type ColumnName struct {
	Schema *Identifier
	Table  *Identifier
	Column Identifier
}

type BindingPower struct {
	L int
	R int
}

type BinaryOp struct {
	Operator tik.Token
	Lhs      Expr
	Rhs      Expr
}

func MakeBinaryOpExpr(
	lhs Expr,
	op tik.Token,
	rhs Expr,
) *BinaryOp {
	return &BinaryOp{
		Operator: op,
		Lhs:      lhs,
		Rhs:      rhs,
	}
}

type CaseExpression struct {
	Operand Expr
	Cases   []WhenThen
	Else    Expr
}

type WhenThen struct {
	When Expr
	Then Expr
}

type Collation struct {
	CollateKeyword Keyword
	Name           Identifier
}

func MakeCollation(
	collateKeyword Keyword,
	name Identifier,
) *Collation {
	return &Collation{
		CollateKeyword: collateKeyword,
		Name:           name,
	}
}
