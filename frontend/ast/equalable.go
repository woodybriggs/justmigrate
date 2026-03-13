package ast

import (
	"maps"
	"reflect"
)

type Equalable interface {
	Eq(otherAny any) bool
}

func As[T any](input any) (*T, bool) {
	if input == nil {
		return nil, false
	}

	switch t := input.(type) {
	case *T:
		return t, true
	case T:
		return &t, true
	default:
		return nil, false
	}
}

func CheckPtr(a, b Equalable) bool {

	if a == nil && b == nil {
		return true
	}

	aIsNil := a == nil || (reflect.ValueOf(a).Kind() == reflect.Ptr && reflect.ValueOf(a).IsNil())
	bIsNil := b == nil || (reflect.ValueOf(b).Kind() == reflect.Ptr && reflect.ValueOf(b).IsNil())

	if aIsNil && bIsNil {
		return true
	}

	if aIsNil || bIsNil {
		return false
	}

	return a.Eq(b)
}

func Check(a, b Equalable) bool {
	return a.Eq(b)
}

func (node *Identifier) Eq(otherAny any) bool {
	other, ok := As[Identifier](otherAny)
	if !ok {
		return false
	}

	if node.Text != other.Text {
		return false
	}

	return true
}

func (node *ParseError) Eq(otherAny any) bool {
	return false
}

func (node *CreateIndex) Eq(otherAny any) bool {
	return false
}

func (node *DropTable) Eq(otherAny any) bool {
	other, ok := As[DropTable](otherAny)
	if !ok {
		return false
	}

	return Check(&node.TableIdentifier, &other.TableIdentifier)
}

func (node *AlterTable) Eq(otherAny any) bool {
	other, ok := As[AlterTable](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.TableIdentifier, other.TableIdentifier) {
		return false
	}

	if !Check(node.Alteration, other.Alteration) {
		return false
	}

	return true
}

func (node *DropColumn) Eq(otherAny any) bool {
	other, ok := As[DropColumn](otherAny)
	if !ok {
		return false
	}

	return Check(&node.ColumnName, &other.ColumnName)
}

func (node *AddColumn) Eq(otherAny any) bool {
	other, ok := As[AddColumn](otherAny)
	if !ok {
		return false
	}

	return Check(&node.ColumnDefinition, &other.ColumnDefinition)
}

func (node *CreateTable) Eq(otherAny any) bool {
	other, ok := As[CreateTable](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Temporary, other.Temporary) {
		return false
	}

	if !CheckPtr(node.TableIdentifier, other.TableIdentifier) {
		return false
	}

	if !CheckPtr(node.TableDefinition, other.TableDefinition) {
		return false
	}

	return true
}

func (node *CatalogObjectIdentifier) Eq(otherAny any) bool {

	if node == nil || otherAny == nil {
		return false
	}

	other, ok := As[CatalogObjectIdentifier](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.SchemaName, other.SchemaName) {
		return false
	}

	if !Check(&node.ObjectName, &other.ObjectName) {
		return false
	}

	return true
}

func (node *TableDefinition) Eq(otherAny any) bool {

	other, ok := As[TableDefinition](otherAny)
	if !ok {
		return false
	}

	if len(node.ColumnDefinitions) != len(other.ColumnDefinitions) {
		return false
	}

	if len(node.TableConstraints) != len(other.TableConstraints) {
		return false
	}

	result := true
	for i := range len(node.ColumnDefinitions) {
		result = result && Check(&node.ColumnDefinitions[i], &other.ColumnDefinitions[i])
	}

	if result == false {
		return false
	}

	for i := range len(node.TableConstraints) {
		result = result && Check(node.TableConstraints[i], other.TableConstraints[i])
	}

	return result
}
func (node *ColumnDefinition) Eq(otherAny any) bool {

	other, ok := As[ColumnDefinition](otherAny)
	if !ok {
		return false
	}

	if !Check(&node.ColumnName, &other.ColumnName) {
		return false
	}

	if !CheckPtr(node.TypeName, other.TypeName) {
		return false
	}

	if len(node.ColumnConstraints) != len(other.ColumnConstraints) {
		return false
	}

	result := true
	for i := range node.ColumnConstraints {
		result = result && Check(node.ColumnConstraints[i], other.ColumnConstraints[i])
	}
	return result
}

func (node *TypeName) Eq(otherAny any) bool {
	other, ok := As[TypeName](otherAny)
	if !ok {
		return false
	}

	if !Check(&node.Name, &other.Name) {
		return false
	}

	if !CheckPtr(node.Arg0, other.Arg0) {
		return false
	}

	if !CheckPtr(node.Arg1, other.Arg1) {
		return false
	}

	return true
}
func (node *ConflictClause) Eq(otherAny any) bool {
	other, ok := As[ConflictClause](otherAny)
	if !ok {
		return false
	}
	return Check(&node.Action, &other.Action)
}

func (node *TableConstraint_Check) Eq(otherAny any) bool {
	other, ok := As[TableConstraint_Check](otherAny)
	if !ok {
		return false
	}

	return Check(node.Expr, other.Expr)
}

func (node *TableConstraint_PrimaryKey) Eq(otherAny any) bool {
	other, ok := As[TableConstraint_PrimaryKey](otherAny)
	if !ok {
		return false
	}

	// assume that named constraints are the same
	if CheckPtr(node.Name, other.Name) {
		return false
	}

	if len(node.IndexedColumns) != len(other.IndexedColumns) {
		return false
	}

	for i := range len(node.IndexedColumns) {
		if !Check(&node.IndexedColumns[i], &other.IndexedColumns[i]) {
			return false
		}
	}

	if !CheckPtr(node.ConflictClause, other.ConflictClause) {
		return false
	}

	return true
}

func (node *TableConstraint_ForeignKey) Eq(otherAny any) bool {
	other, ok := As[TableConstraint_ForeignKey](otherAny)
	if !ok {
		return false
	}

	thisPairs := node.pairColumns()
	otherPairs := other.pairColumns()

	if len(thisPairs) != len(otherPairs) {
		return false
	}

	for i := range len(thisPairs) {
		if !Check(&thisPairs[i], &otherPairs[i]) {
			return false
		}
	}

	if !Check(&node.FkClause, &other.FkClause) {
		return false
	}

	return true
}

func (node *IndexedColumn) Eq(otherAny any) bool {
	other, ok := As[IndexedColumn](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Subject, other.Subject) {
		return false
	}

	if !Check(node.Collation, other.Collation) {
		return false
	}

	if !CheckPtr(node.Collation, other.Collation) {
		return false
	}

	if CheckPtr(node.Order, other.Order) {
		return false
	}

	return true
}

func (node *ForeignKeyClause) Eq(otherAny any) bool {

	other, ok := As[ForeignKeyClause](otherAny)
	if !ok {
		return false
	}

	if len(node.ForeignColumns) != len(other.ForeignColumns) {
		return false
	}
	if len(node.Actions) != len(other.Actions) {
		return false
	}

	if !Check(&node.ForeignTable, &other.ForeignTable) {
		return false
	}

	if !Check(IdentifierList(node.ForeignColumns).ToSet(), IdentifierList(other.ForeignColumns).ToSet()) {
		return false
	}

	return true
}

func (node *NoAction) Eq(otherAny any) bool {
	_, ok := As[NoAction](otherAny)
	return ok
}

func (node *Restrict) Eq(otherAny any) bool {
	_, ok := As[Restrict](otherAny)
	return ok
}

func (node *SetNull) Eq(otherAny any) bool {
	_, ok := As[SetNull](otherAny)
	return ok
}

func (node *SetDefault) Eq(otherAny any) bool {
	_, ok := As[SetDefault](otherAny)
	return ok
}

func (node *Cascade) Eq(otherAny any) bool {
	_, ok := As[Cascade](otherAny)
	return ok
}

func (node *ConstraintName) Eq(otherAny any) bool {
	other, ok := As[ConstraintName](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(&node.Name, &other.Name) {
		return false
	}

	return true
}

func (node *ColumnConstraint_PrimaryKey) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_PrimaryKey](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Order, other.Order) {
		return false
	}

	if !CheckPtr(node.AutoIncrement, other.AutoIncrement) {
		return false
	}

	if !CheckPtr(node.ConflictClause, other.ConflictClause) {
		return false
	}

	return true
}

type IdentifierSet map[string]struct{}

func (this IdentifierSet) Eq(otherAny any) bool {

	other, ok := otherAny.(IdentifierSet)
	if !ok {
		return false
	}

	if len(this) != len(other) {
		return false
	}

	for thisKey := range maps.Keys(this) {
		if _, hasKey := other[thisKey]; !hasKey {
			return false
		}
	}

	return true
}

type IdentifierList []Identifier

func (l IdentifierList) ToSet() (result IdentifierSet) {
	result = IdentifierSet{}
	for _, str := range l {
		result[str.Text] = struct{}{}
	}
	return result
}

func (node *ColumnConstraint_ForeignKey) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_ForeignKey](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	if !Check(&node.FkClause.ForeignTable, &other.FkClause.ForeignTable) {
		return false
	}

	if !Check(
		IdentifierList(node.FkClause.ForeignColumns).ToSet(),
		IdentifierList(other.FkClause.ForeignColumns).ToSet(),
	) {
		return false
	}

	if !CheckPtr(node.FkClause.MatchName, other.FkClause.MatchName) {
		return false
	}

	if !CheckPtr(node.FkClause.Deferrable, other.FkClause.Deferrable) {
		return false
	}

	return true
}

func (node *ColumnConstraint_NotNull) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_NotNull](otherAny)
	if !ok {
		return false
	}

	return CheckPtr(node.Name, other.Name)
}

func (node *ColumnConstraint_Default) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_Default](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	return Check(node.Default, other.Default)
}

func (node *ColumnConstraint_Generated) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_Generated](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	if !Check(node.AsExpr, other.AsExpr) {
		return false
	}

	return true
}

func (node *ColumnConstraint_Check) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_Check](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	if !Check(node.CheckExpr, other.CheckExpr) {
		return false
	}

	return true
}

func (node *ColumnConstraint_Collate) Eq(otherAny any) bool {
	other, ok := As[ColumnConstraint_Collate](otherAny)
	if !ok {
		return false
	}

	if !Check(&node.CollationName, &other.CollationName) {
		return false
	}

	return true
}

func (node *ColumnConstraint_Unique) Eq(otherAny any) bool {
	_, ok := As[ColumnConstraint_Unique](otherAny)
	return ok
}

func (node ExprList) Eq(otherAny any) bool {
	other, ok := otherAny.(ExprList)
	if !ok {
		return false
	}

	if len(node) != len(other) {
		return false
	}

	for i := range len(node) {
		if !Check(node[i], other[i]) {
			return false
		}
	}

	return true
}

func (node *Keyword) Eq(otherAny any) bool {

	other, ok := As[Keyword](otherAny)
	if !ok {
		return false
	}

	if node == nil || other == nil {
		return false
	}

	if node.Text != other.Text {
		return false
	}

	return true
}

func (node *Collation) Eq(otherAny any) bool {
	other, ok := As[Collation](otherAny)
	if !ok {
		return false
	}

	if !Check(&node.Name, &other.Name) {
		return false
	}

	return true
}

func (node WhenThen) Eq(otherAny any) bool {
	other, ok := otherAny.(WhenThen)
	if !ok {
		return false
	}

	if !Check(node.When, other.When) {
		return false
	}

	if !Check(node.Then, other.Then) {
		return false
	}

	return true
}

func (node *CaseExpression) Eq(otherAny any) bool {

	other, ok := As[CaseExpression](otherAny)
	if !ok {
		return false
	}

	if !Check(node.Operand, other.Operand) {
		return false
	}

	if !Check(node.Else, other.Else) {
		return false
	}

	if len(node.Cases) != len(other.Cases) {
		return false
	}

	for i := range len(node.Cases) {
		if !Check(node.Cases[i], other.Cases[i]) {
			return false
		}
	}
	return true
}

func (node *BinaryOp) Eq(otherAny any) bool {
	other, ok := As[BinaryOp](otherAny)
	if !ok {
		return false
	}

	if node.Operator.Kind != other.Operator.Kind {
		return false
	}

	if !Check(node.Lhs, other.Lhs) {
		return false
	}

	if !Check(node.Rhs, other.Rhs) {
		return false
	}

	return true
}

func (node *ColumnName) Eq(otherAny any) bool {
	other, ok := As[ColumnName](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.Schema, other.Schema) {
		return false
	}

	if !CheckPtr(node.Table, other.Schema) {
		return false
	}

	if !Check(&node.Column, &other.Column) {
		return false
	}

	return true
}

func (node *FunctionCall) Eq(otherAny any) bool {
	other, ok := As[FunctionCall](otherAny)
	if !ok {
		return false
	}

	if !Check(&node.Name, &other.Name) {
		return false
	}

	if len(node.Args) != len(other.Args) {
		return false
	}

	if !Check(node.Args, other.Args) {
		return false
	}

	return true
}

func (node *LiteralString) Eq(otherAny any) bool {
	other, ok := As[LiteralString](otherAny)
	if !ok {
		return false
	}
	return node.Value == other.Value
}

func (node *LiteralFloat) Eq(otherAny any) bool {
	other, ok := As[LiteralFloat](otherAny)
	if !ok {
		// we check the text value here as to not compare floats.
		return false
	}
	return other.Token.Text == node.Token.Text
}

func (node *LiteralSignedInteger) Eq(otherAny any) bool {
	other, ok := As[LiteralSignedInteger](otherAny)
	if !ok {
		return false
	}

	if node.Value != other.Value {
		return false
	}

	return true
}

func (node *LiteralUnsignedInteger) Eq(otherAny any) bool {
	other, ok := As[LiteralUnsignedInteger](otherAny)
	if !ok {
		return false
	}

	if node.Value != other.Value {
		return false
	}

	return true
}

func (node *LiteralBoolean) Eq(otherAny any) bool {
	other, ok := As[LiteralBoolean](otherAny)
	if !ok {
		return false
	}

	if node.Value != other.Value {
		return false
	}

	return true
}

func (node *LiteralNull) Eq(otherAny any) bool {
	_, ok := As[LiteralNull](otherAny)
	return ok
}

func (this IdentifierPair) Eq(otherAny any) bool {
	other, ok := otherAny.(IdentifierPair)
	if !ok {
		return false
	}
	if !Check(&this.A, &other.A) {
		return false
	}
	if !Check(&this.B, &other.B) {
		return false
	}
	return true
}

func (node *ForeignKeyDeleteAction) Eq(otherAny any) bool {
	other, ok := As[ForeignKeyDeleteAction](otherAny)
	if !ok {
		return false
	}

	if !Check(node.Action, other.Action) {
		return false
	}

	return true
}

func (node *ForeignKeyUpdateAction) Eq(otherAny any) bool {
	other, ok := As[ForeignKeyUpdateAction](otherAny)
	if !ok {
		return false
	}

	if !Check(node.Action, other.Action) {
		return false
	}

	return true
}

func (node *ForeignKeyDeferrable) Eq(otherAny any) bool {

	other, ok := As[ForeignKeyDeferrable](otherAny)
	if !ok {
		return false
	}

	if !CheckPtr(node.NotKeyword, other.NotKeyword) {
		return false
	}

	if !CheckPtr(node.InitiallyKeyword, other.InitiallyKeyword) {
		return false
	}

	if !CheckPtr(node.Deferrable, other.Deferrable) {
		return false
	}

	return true
}
