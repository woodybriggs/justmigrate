package main

import (
	"strings"
)

type AstNode interface {
	node()
}

type Statement interface {
	AstNode
	nodeStatement()
}

type TableConstraint interface {
	AstNode
	nodeTableConstraint()
}

type ColumnConstraint interface {
	AstNode
	nodeColumnConstraint()
}

type Constraint interface {
	AstNode
	TableConstraint
	ColumnConstraint
}

type Expr interface {
	AstNode
	nodeExpression()
}

type BinaryOperator interface {
	AstNode
	Expr
	nodeBinaryOperator()
	bindingPower() BindingPower
	SetLhs(expr Expr)
	SetRhs(expr Expr)
}

type Literal interface {
	AstNode
	Expr
	nodeLiteral()
}

type Identifier interface {
	AstNode
	Expr
	nodeIdentifier()
}

func (t *Token) node()            {}
func (t *Token) nodeExpression()  {}
func (t *Token) nodeIdentifier()  {}
func (t *Token) nodePragmaValue() {}

type Error struct {
	err            error
	OffendingToken Token
	TokenizerData
	SourceCode
}

func (node *Error) node()                        {}
func (node *Error) nodeIdentifier()              {}
func (node *Error) nodeStatement()               {}
func (node *Error) nodeConstraint()              {}
func (node *Error) nodeColumnConstraint()        {}
func (node *Error) nodeTableConstraint()         {}
func (node *Error) nodeLiteral()                 {}
func (node *Error) nodeExpression()              {}
func (node *Error) nodeBinaryOperator()          {}
func (node *Error) nodeForeignKeyAction()        {}
func (node *Error) nodeForeignKeyActionTrigger() {}
func (node *Error) nodePragmaValue()             {}

func NewError(err error, offendingToken Token, tokenizerData TokenizerData, sourceCode SourceCode) *Error {
	return &Error{
		err:            err,
		OffendingToken: offendingToken,
		TokenizerData:  tokenizerData,
		SourceCode:     sourceCode,
	}
}

func (node *Error) Error() string {
	return node.err.Error()
}

type Pragma struct {
	Name  AstNode
	Value PragmaValue
}

func (node *Pragma) node()          {}
func (node *Pragma) nodeStatement() {}

type PragmaValue interface {
	AstNode
	nodePragmaValue()
}

type BeginTransaction struct{}

func (node *BeginTransaction) node()          {}
func (node *BeginTransaction) nodeStatement() {}

type CommitTransaction struct{}

func (node *CommitTransaction) node()          {}
func (node *CommitTransaction) nodeStatement() {}

type Select struct{}

func (node *Select) node()          {}
func (node *Select) nodeStatement() {}

type CreateTrigger struct{}

func (node *CreateTrigger) node()          {}
func (node *CreateTrigger) nodeStatement() {}

type CreateTable struct {
	IsTemporary bool
	IfNotExist  bool

	TableIdentifier AstNode
	TableDefinition AstNode

	TableOptions AstNode
}

func (node *CreateTable) node()          {}
func (node *CreateTable) nodeStatement() {}

type CreateVirtualTable struct {
	IfNotExist      bool
	TableIdentifier AstNode

	ModuleName Identifier
	ModuleArgs []string
}

func (node *CreateVirtualTable) node()          {}
func (node *CreateVirtualTable) nodeStatement() {}

type CreateIndex struct {
	Unique          bool
	IfNotExists     bool
	IndexIdentifier AstNode
	OnTable         Identifier
	IndexedColumns  []AstNode
	WhereExpr       Expr
}

func (node *CreateIndex) node()          {}
func (node *CreateIndex) nodeStatement() {}

type IndexedColumn struct {
	Subject       AstNode
	CollationName AstNode
	Order         AstNode
}

func (node *IndexedColumn) node() {}

type CreateView struct {
	IfNotExists    bool
	ViewIdentifier AstNode
	Columns        []Identifier
	AsSelect       AstNode
}

func (node *CreateView) node()          {}
func (node *CreateView) nodeStatement() {}

type IfNotExists struct{}

func (node *IfNotExists) node() {}

type CatalogObjectIdentifier struct {
	SchemaName AstNode
	ObjectName AstNode
}

func (node *CatalogObjectIdentifier) node() {}

func (node *CatalogObjectIdentifier) FullyQualifiedName() string {

	result := []string{}

	if schemaName, ok := node.SchemaName.(Identifier); ok && schemaName != nil {
		result = append(result, schemaName.(*Token).Text)
	}

	if objectName, ok := node.ObjectName.(Identifier); ok && objectName != nil {
		result = append(result, objectName.(*Token).Text)
	}

	return strings.Join(result, ".")
}

type TableDefinition struct {
	// AsSelect *SelectStatement
	ColumnDefinitions []AstNode
	TableConstraints  []AstNode
}

func (node *TableDefinition) node() {}

type TableOptions struct {
	Strict       bool
	WithoutRowId bool
}

func (node *TableOptions) node() {}

type ColumnDefinition struct {
	ColumnName        AstNode
	TypeName          AstNode
	ColumnConstraints []AstNode
}

func (node *ColumnDefinition) node() {}

type TypeName struct {
	TypeName AstNode
}

func (node *TypeName) node() {}

// type Identifier struct {
// 	Identifier Token
// }

// func (node *Identifier) node()     {}
// func (node *Identifier) nodeExpression() {}

type TableConstraint_PrimaryKey struct {
	Name           AstNode
	IndexedColumns []AstNode
	ConflictClause AstNode
}

func (node *TableConstraint_PrimaryKey) node()                {}
func (node *TableConstraint_PrimaryKey) nodeTableConstraint() {}

type TableConstraint_ForeignKey struct {
	Name     AstNode
	Columns  []AstNode
	FkClause AstNode
}

func (node *TableConstraint_ForeignKey) node()                {}
func (node *TableConstraint_ForeignKey) nodeTableConstraint() {}

type ForeignKeyDeferrable int

func (node ForeignKeyDeferrable) node() {}

const (
	ForeignKeyDeferrable_Immediate ForeignKeyDeferrable = iota
	ForeignKeyDeferrable_Deferred
)

type ForeignKeyClause struct {
	ForeignTable   Identifier
	ForeignColumns []AstNode
	Actions        []ForeignKeyActionTrigger
	MatchName      AstNode
	Deferrable     AstNode
}

func (node *ForeignKeyClause) node() {}

type ForeignKeyActionTrigger interface {
	AstNode
	nodeForeignKeyActionTrigger()
}

type OnDelete struct {
	Action ForeignKeyAction
}

func (node *OnDelete) node()                        {}
func (node *OnDelete) nodeForeignKeyActionTrigger() {}

type OnUpdate struct {
	Action ForeignKeyAction
}

func (node *OnUpdate) node()                        {}
func (node *OnUpdate) nodeForeignKeyActionTrigger() {}

type ForeignKeyAction interface {
	AstNode
	nodeForeignKeyAction()
}

type NoAction struct{}

func (node *NoAction) node()                 {}
func (node *NoAction) nodeForeignKeyAction() {}

type Restrict struct{}

func (node *Restrict) node()                 {}
func (node *Restrict) nodeForeignKeyAction() {}

type SetNull struct{}

func (node *SetNull) node()                 {}
func (node *SetNull) nodeForeignKeyAction() {}

type SetDefault struct{}

func (node *SetDefault) node()                 {}
func (node *SetDefault) nodeForeignKeyAction() {}

type Cascade struct{}

func (node *Cascade) node()                 {}
func (node *Cascade) nodeForeignKeyAction() {}

type OrderBy interface {
	AstNode
}

type ColumnConstraint_PrimaryKey struct {
	Name           AstNode
	Order          AstNode
	ConflictClause AstNode
	AutoIncrement  bool
}

func (node *ColumnConstraint_PrimaryKey) node()                 {}
func (node *ColumnConstraint_PrimaryKey) nodeColumnConstraint() {}

type ColumnConstraint_Unique struct {
	Name AstNode
}

func (node *ColumnConstraint_Unique) node()                 {}
func (node *ColumnConstraint_Unique) nodeColumnConstraint() {}

type ColumnConstraint_Collate struct {
	Name    AstNode
	Collate AstNode
}

func (node *ColumnConstraint_Collate) node()                 {}
func (node *ColumnConstraint_Collate) nodeColumnConstraint() {}

type ColumnConstraint_NotNull struct {
	Name AstNode
}

func (node *ColumnConstraint_NotNull) node()                 {}
func (node *ColumnConstraint_NotNull) nodeColumnConstraint() {}

type ColumnConstraint_Default struct {
	Name    AstNode
	Default Expr
}

func (node *ColumnConstraint_Default) node()                 {}
func (node *ColumnConstraint_Default) nodeColumnConstraint() {}

type GeneratedColumnStorage struct {
	Token Token
	Value GeneratedColumnStorageValue
}

func (node *GeneratedColumnStorage) node() {}

type GeneratedColumnStorageValue int

const (
	GeneratedColumnStorageValue_Virtual GeneratedColumnStorageValue = iota
	GeneratedColumnStorageValue_Stored
)

type ColumnConstraint_Generated struct {
	Name    AstNode
	As      Expr
	Storage AstNode
}

func (node *ColumnConstraint_Generated) node()                 {}
func (node *ColumnConstraint_Generated) nodeColumnConstraint() {}

type Constraint_Check struct {
	Name  AstNode
	Check Expr
}

func (node *Constraint_Check) node()                 {}
func (node *Constraint_Check) nodeColumnConstraint() {}
func (node *Constraint_Check) nodeTableConstraint()  {}

type ExprList []AstNode

func (node ExprList) node()           {}
func (node ExprList) nodeExpression() {}

type LiteralNull struct {
	Token Token
}

func (node *LiteralNull) astnode()        {}
func (node *LiteralNull) nodeExpression() {}
func (node *LiteralNull) nodeLiteral()    {}

type LiteralBoolean struct {
	Token Token
	Value AstNode
}

func (node *LiteralBoolean) node()            {}
func (node *LiteralBoolean) nodeExpression()  {}
func (node *LiteralBoolean) nodeLiteral()     {}
func (node *LiteralBoolean) nodePragmaValue() {}

type Float float64

func (f Float) node() {}

type Integer int64

func (f Integer) node() {}

type Boolean bool

func (f Boolean) node() {}

type LiteralNumber struct {
	Token Token
	Value AstNode
}

func (node *LiteralNumber) node()            {}
func (node *LiteralNumber) nodeExpression()  {}
func (node *LiteralNumber) nodeLiteral()     {}
func (node *LiteralNumber) nodePragmaValue() {}

type LiteralString struct {
	Token Token
	Value string
}

func (node *LiteralString) node()            {}
func (node *LiteralString) nodeExpression()  {}
func (node *LiteralString) nodeLiteral()     {}
func (node *LiteralString) nodePragmaValue() {}

type UnaryOperator struct {
	Operator Token
	Rhs      Expr
}

func (node *UnaryOperator) node()           {}
func (node *UnaryOperator) nodeExpression() {}

type FunctionCall struct {
	Name Token
	Args ExprList
}

func (node *FunctionCall) node()           {}
func (node *FunctionCall) nodeExpression() {}

type ColumnName struct {
	Schema Identifier
	Table  Identifier
	Column Identifier
}

func (node *ColumnName) node()           {}
func (node *ColumnName) nodeExpression() {}

type BindingPower struct {
	L int
	R int
}

type BinaryOp struct {
	Lhs Expr
	Rhs Expr
}

func (node *Error) SetLhs(expr Expr) {}
func (node *Error) SetRhs(expr Expr) {}

func (node *Error) bindingPower() BindingPower {
	return BindingPower{0, 0}
}

type AddOp BinaryOp

func (node *AddOp) node()               {}
func (node *AddOp) nodeExpression()     {}
func (node *AddOp) nodeBinaryOperator() {}
func (node *AddOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *AddOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *AddOp) bindingPower() BindingPower {
	return BindingPower{60, 61}
}

type SubOp BinaryOp

func (node *SubOp) node()               {}
func (node *SubOp) nodeExpression()     {}
func (node *SubOp) nodeBinaryOperator() {}
func (node *SubOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *SubOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *SubOp) bindingPower() BindingPower {
	return BindingPower{60, 61}
}

type MulOp BinaryOp

func (node *MulOp) node()               {}
func (node *MulOp) nodeExpression()     {}
func (node *MulOp) nodeBinaryOperator() {}
func (node *MulOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *MulOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *MulOp) bindingPower() BindingPower {
	return BindingPower{120, 121}
}

type DivOp BinaryOp

func (node *DivOp) node()               {}
func (node *DivOp) nodeExpression()     {}
func (node *DivOp) nodeBinaryOperator() {}
func (node *DivOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *DivOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *DivOp) bindingPower() BindingPower {
	return BindingPower{120, 121}
}

type GteOp BinaryOp

func (node *GteOp) node()               {}
func (node *GteOp) nodeExpression()     {}
func (node *GteOp) nodeBinaryOperator() {}
func (node *GteOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *GteOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *GteOp) bindingPower() BindingPower {
	return BindingPower{50, 50}
}

type InOp BinaryOp

func (node *InOp) node()               {}
func (node *InOp) nodeExpression()     {}
func (node *InOp) nodeBinaryOperator() {}
func (node *InOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *InOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *InOp) bindingPower() BindingPower {
	return BindingPower{40, 40}
}

type EquivOp BinaryOp

func (node *EquivOp) node()               {}
func (node *EquivOp) nodeExpression()     {}
func (node *EquivOp) nodeBinaryOperator() {}
func (node *EquivOp) SetLhs(expr Expr) {
	node.Lhs = expr
}
func (node *EquivOp) SetRhs(expr Expr) {
	node.Rhs = expr
}
func (node *EquivOp) bindingPower() BindingPower {
	return BindingPower{40, 40}
}

type CaseExpression struct {
	Operand Expr
	Cases   []WhenThen
	Else    Expr
}

func (node *CaseExpression) node()           {}
func (node *CaseExpression) nodeExpression() {}

type WhenThen struct {
	When Expr
	Then Expr
}
