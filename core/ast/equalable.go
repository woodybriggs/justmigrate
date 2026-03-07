package ast

import (
	"maps"
	"slices"
)

type Equalable[T any] interface {
	Eq(other T) bool
}

func CheckPtr[T Equalable[T]](a, b T) bool {
	aIsNil := any(a) == nil
	bIsNil := any(b) == nil

	if aIsNil && bIsNil {
		return true
	}
	if aIsNil || bIsNil {
		return false
	}

	return a.Eq(b)
}

func Check[T Equalable[T]](a, b T) bool {
	return a.Eq(b)
}

func (node *CreateIndex) Eq(otherStatement Statement) bool {
	return false
}

func (node *DropTable) Eq(otherStatement Statement) bool {
	other, ok := otherStatement.(*DropTable)
	if !ok {
		return false
	}

	return Check(&node.TableIdentifier, &other.TableIdentifier)
}

func (node *AlterTable) Eq(otherStatement Statement) bool {
	other, ok := otherStatement.(*AlterTable)
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

func (node *DropColumn) Eq(otherAlteration TableAlteration) bool {
	other, ok := otherAlteration.(*DropColumn)
	if !ok {
		return false
	}

	return Check(node.ColumnName.AsExpr(), other.ColumnName.AsExpr())
}

func (node *AddColumn) Eq(otherAlteration TableAlteration) bool {
	other, ok := otherAlteration.(*AddColumn)
	if !ok {
		return false
	}

	return Check(&node.ColumnDefinition, &other.ColumnDefinition)
}

func (node *CreateTable) Eq(otherStatement Statement) bool {
	other, ok := otherStatement.(*CreateTable)
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

func (node *CatalogObjectIdentifier) Eq(other *CatalogObjectIdentifier) bool {

	if other == nil {
		return false
	}

	result := true
	if node.SchemaName != nil && other.SchemaName != nil {
		result = result && node.SchemaName.Eq(other.SchemaName.AsExpr())
	} else if node.SchemaName != nil || other.SchemaName != nil {
		return false
	}

	result = result && node.ObjectName.Eq(other.ObjectName.AsExpr())

	return result
}

func (node *TableDefinition) Eq(other *TableDefinition) bool {

	if len(node.ColumnDefinitions) != len(other.ColumnDefinitions) {
		return false
	}

	if len(node.TableConstraints) != len(other.TableConstraints) {
		return false
	}

	result := true
	for i := range len(node.ColumnDefinitions) {
		a, b := node.ColumnDefinitions[i], other.ColumnDefinitions[i]
		result = result && a.Eq(&b)
	}

	if result == false {
		return false
	}

	for i := range len(node.TableConstraints) {
		a, b := node.TableConstraints[i], other.TableConstraints[i]
		result = result && a.Eq(b)
	}

	return result
}
func (node *ColumnDefinition) Eq(other *ColumnDefinition) bool {
	if !Check(node.ColumnName.AsExpr(), other.ColumnName.AsExpr()) {
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
		a, b := node.ColumnConstraints[i], other.ColumnConstraints[i]
		result = result && a.Eq(b)
	}
	return result
}

func (node *TypeName) Eq(other *TypeName) bool {
	if !node.Name.Eq(other.Name.AsExpr()) {
		return false
	}
	return true
}
func (node *ConflictClause) Eq(other *ConflictClause) bool {
	return node.Action.Eq(&other.Action)
}

func (node *TableConstraint_Check) Eq(other TableConstraint) bool {

	if otherCheck, ok := other.(*TableConstraint_Check); ok {
		// Assume that named constraints are equivilent
		if node.Name.Eq(otherCheck.Name) {
			return true
		}

		// Otherwise we need to know if the Expression is "equivilent"
		return node.Expr.Eq(otherCheck.Expr)
	}

	return true
}

func (node *TableConstraint_PrimaryKey) Eq(other TableConstraint) bool {
	if other, ok := other.(*TableConstraint_PrimaryKey); ok {

		if len(node.IndexedColumns) != len(other.IndexedColumns) {
			return false
		}

		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		for i, column := range node.IndexedColumns {
			otherColumn := &other.IndexedColumns[i]
			result = result && column.Eq(otherColumn)
		}

		if node.ConflictClause != nil && other.ConflictClause != nil {
			result = result && node.ConflictClause.Eq(other.ConflictClause)
		} else if node.ConflictClause != nil || other.ConflictClause != nil {
			return false
		}

		return true
	}
	return false
}

func (node *TableConstraint_ForeignKey) Eq(otherTableConstraint TableConstraint) bool {
	other, ok := otherTableConstraint.(*TableConstraint_ForeignKey)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	thisPairs := node.pairColumns()
	otherPairs := other.pairColumns()

	if len(thisPairs) != len(otherPairs) {
		return false
	}

	result := true

	for i := range len(thisPairs) {
		thisPair, otherPair := thisPairs[i], otherPairs[i]
		result = result && thisPair.Eq(otherPair)
	}

	result = result && node.FkClause.Eq(&other.FkClause)
	return result
}

func (node *IndexedColumn) Eq(other *IndexedColumn) bool {
	result := true

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

	return result
}

func (node *ForeignKeyClause) Eq(other *ForeignKeyClause) bool {
	if len(node.ForeignColumns) != len(other.ForeignColumns) {
		return false
	}
	if len(node.Actions) != len(other.Actions) {
		return false
	}

	result := true
	result = result && node.ForeignTable.Eq(&other.ForeignTable)

	cmp := func(a, b Identifier) int {
		if a.Text < b.Text {
			return -1
		}
		if a.Text > b.Text {
			return 1
		}
		return 0
	}

	aCols := slices.SortedFunc(slices.Values(node.ForeignColumns), cmp)
	bCols := slices.SortedFunc(slices.Values(other.ForeignColumns), cmp)

	for i, foreignCol := range aCols {
		result = result && foreignCol.Eq(bCols[i].AsExpr())
	}

	return result
}

func (node *NoAction) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*NoAction)
	return ok
}

func (node *Restrict) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*Restrict)
	return ok
}

func (node *SetNull) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*SetNull)
	return ok
}

func (node *SetDefault) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*SetDefault)
	return ok
}

func (node *Cascade) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*Cascade)
	return ok
}

func (node *ConstraintName) Eq(other *ConstraintName) bool {
	if node == nil || other == nil {
		return false
	}
	return CheckPtr(node.Name.AsExpr(), other.Name.AsExpr())
}

func (node *ColumnConstraint_PrimaryKey) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_PrimaryKey); ok {
		result := true

		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		return result
	}
	return false
}

type IdentifierSet map[string]struct{}

func (this IdentifierSet) Eq(other IdentifierSet) bool {

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

func (node *ColumnConstraint_ForeignKey) Eq(otherConstraint ColumnConstraint) bool {
	other, ok := otherConstraint.(*ColumnConstraint_ForeignKey)
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

	if !CheckPtr(
		node.FkClause.MatchName.AsExpr(),
		other.FkClause.MatchName.AsExpr(),
	) {
		return false
	}

	if !CheckPtr(
		node.FkClause.Deferrable,
		other.FkClause.Deferrable,
	) {
		return false
	}

	return true
}

func (node *ColumnConstraint_NotNull) Eq(otherColumnConstraint ColumnConstraint) bool {
	other, ok := otherColumnConstraint.(*ColumnConstraint_NotNull)
	if !ok {
		return false
	}

	return CheckPtr(node.Name, other.Name)
}

func (node *ColumnConstraint_Default) Eq(otherColumnConstraint ColumnConstraint) bool {
	other, ok := otherColumnConstraint.(*ColumnConstraint_Default)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	return Check(node.Default, other.Default)
}

func (node *ColumnConstraint_Generated) Eq(otherColumnConstraint ColumnConstraint) bool {
	other, ok := otherColumnConstraint.(*ColumnConstraint_Generated)
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

func (node *ColumnConstraint_Check) Eq(otherColumnConstraint ColumnConstraint) bool {
	other, ok := otherColumnConstraint.(*ColumnConstraint_Check)
	if !ok {
		return false
	}

	if !CheckPtr(node.Name, other.Name) {
		return false
	}

	return Check(node.CheckExpr, other.CheckExpr)
}

// @note(woody): fix up ^see above
func (node *ColumnConstraint_Collate) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_Collate); ok {
		result := true

		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		result = result && node.CollationName.Eq(other.CollationName.AsExpr())
		return result
	}
	return false
}

func (node *ColumnConstraint_Unique) Eq(other ColumnConstraint) bool {
	_, ok := other.(*ColumnConstraint_Unique)
	return ok
}

func (node ExprList) Eq(otherExpr Expr) bool {
	other, ok := otherExpr.(ExprList)
	if !ok {
		return false
	}

	if len(node) != len(other) {
		return false
	}

	result := true
	for i := range len(node) {
		a, b := node[i], other[i]
		result = result && a.Eq(b)
	}

	return result
}

func (node *Keyword) Eq(other *Keyword) bool {

	if node == nil && other == nil {
		return true
	}

	if node != nil && other != nil {
		return node.Kind == other.Kind
	}

	return false
}

func (node *Collation) Eq(other *Collation) bool {
	if !Check(node.Name.AsExpr(), other.Name.AsExpr()) {
		return false
	}
	return true
}

func (node WhenThen) Eq(other WhenThen) bool {
	return Check(node.When, other.When) && Check(node.Then, other.Then)
}

func (node *CaseExpression) Eq(otherExpr Expr) bool {

	other, ok := otherExpr.(*CaseExpression)
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

	result := true
	for i := range len(node.Cases) {
		a, b := node.Cases[i], other.Cases[i]
		result = result && a.Eq(b)
	}
	return true
}

func (node *BinaryOp) Eq(otherExpr Expr) bool {
	other, ok := otherExpr.(*BinaryOp)
	if !ok {
		return false
	}

	if node.Operator.Kind != other.Operator.Kind {
		return false
	}

	return Check(node.Lhs, other.Lhs) && Check(node.Rhs, other.Rhs)
}

func (node *ColumnName) Eq(otherExpr Expr) bool {
	other, ok := otherExpr.(*ColumnName)
	if !ok {
		return false
	}

	if !CheckPtr(node.Schema.AsExpr(), other.Schema.AsExpr()) {
		return false
	}

	if !CheckPtr(node.Table.AsExpr(), other.Schema.AsExpr()) {
		return false
	}

	return Check(node.Column.AsExpr(), other.Column.AsExpr())
}

func (node *FunctionCall) Eq(otherExpr Expr) bool {
	other, ok := otherExpr.(*FunctionCall)
	if !ok {
		return false
	}

	if !Check(node.Name.AsExpr(), other.Name.AsExpr()) {
		return false
	}

	if len(node.Args) != len(other.Args) {
		return false
	}

	return node.Args.Eq(other.Args)
}

func (node *LiteralString) Eq(otherExpr Expr) bool {
	other, ok := otherExpr.(*LiteralString)
	if !ok {
		return false
	}
	return node.Value == other.Value
}

func (node *LiteralFloat) Eq(other Expr) bool {
	if otherFloat, ok := other.(*LiteralFloat); ok {
		// we check the text value here as to not compare floats.
		return otherFloat.Token.Text == node.Token.Text
	}
	return false
}

func (node *LiteralSignedInteger) Eq(other Expr) bool {
	if otherNumber, ok := other.(*LiteralSignedInteger); ok {
		return otherNumber.Token.Text == node.Token.Text
	}
	return false
}

func (node *LiteralUnsignedInteger) Eq(other Expr) bool {
	if otherNumber, ok := other.(*LiteralUnsignedInteger); ok {
		return otherNumber.Value == node.Value
	}

	return false
}

func (node *LiteralBoolean) Eq(other Expr) bool {
	if otherBool, ok := other.(*LiteralBoolean); ok {
		return node.Value == otherBool.Value
	}
	return false
}

func (node *LiteralNull) Eq(other Expr) bool {
	if _, ok := other.(*LiteralNull); ok {
		return true
	}
	return false
}

func (this IdentifierPair) Eq(other IdentifierPair) bool {
	return this.A.Eq(other.A.AsExpr()) && this.B.Eq(other.B.AsExpr())
}

func (node *ForeignKeyDeleteAction) Eq(otherFkAction ForeignKeyAction) bool {
	other, ok := otherFkAction.(*ForeignKeyDeleteAction)
	if !ok {
		return false
	}

	return Check(node.Action, other.Action)
}

func (node *ForeignKeyUpdateAction) Eq(otherFkAction ForeignKeyAction) bool {
	other, ok := otherFkAction.(*ForeignKeyUpdateAction)
	if !ok {
		return false
	}

	return Check(node.Action, other.Action)
}

func (node *ForeignKeyDeferrable) Eq(other *ForeignKeyDeferrable) bool {
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
