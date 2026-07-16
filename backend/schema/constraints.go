package schema

import (
	"errors"
	"fmt"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
	"woodybriggs/justmigrate/frontend/token"
)

type PrimaryKey struct {
	Node interface {
		Accept(ast.Visitor)
		ast.Equalable
	}

	Name            *ast.Identifier
	IndexedColumns  []ast.IndexedColumn
	ResolvedColumns []ColumnLike
	Order           Order
	ConflictClause  ConflictAction
}

func (pk *PrimaryKey) Eq(otherPK any) bool {
	other, ok := otherPK.(*PrimaryKey)
	if !ok {
		return false
	}
	return pk.Node.Eq(other.Node)
}

func PrimaryKeyFromTableConstraintAst(table *Table, constraint *ast.TableConstraint_PrimaryKey) (*PrimaryKey, error) {
	result := &PrimaryKey{
		Node:            constraint,
		IndexedColumns:  constraint.IndexedColumns,
		ResolvedColumns: make([]ColumnLike, len(constraint.IndexedColumns)),
		ConflictClause:  ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		result.Name = &constraint.Name.Name
	}

	for i, indexedCol := range constraint.IndexedColumns {
		ident, ok := indexedCol.Subject.(*ast.Identifier)
		if !ok {
			result.ResolvedColumns[i] = nil
			continue
		}

		col, ok := table.Columns[ident.String()]
		if !ok {
			result.ResolvedColumns[i] = nil
			continue
		}

		result.ResolvedColumns[i] = col
	}

	return result, nil
}

func PrimaryKeyFromColumnConstraintAst(colLike ColumnLike, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_PrimaryKey) error {

	if constraints.PK != nil {
		// primary key already declared on column error
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.PrimaryKeyword.FileLoc).
			WithMessage("column has already delcared a foreign key constraint").
			WithLabels(
				report.LabelFromKeyword(constraints.PK.Node.(*ast.ColumnConstraint_PrimaryKey).PrimaryKeyword, "previous primary key constraint declared here"),
			)
	}

	constraints.PK = &PrimaryKey{
		Node: constraint,
		IndexedColumns: []ast.IndexedColumn{
			{
				Subject: colLike.GetName(),
				Order:   constraint.Order,
			},
		},
		ConflictClause: ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		constraints.PK.Name = &constraint.Name.Name
	}

	return nil
}

func ConflictClauseFromAst(conflictClause *ast.ConflictClause) ConflictAction {
	if conflictClause != nil {
		switch conflictClause.Action.Kind {
		case token.TokenKind_Keyword_ROLLBACK:
			return ConflictActionRollback
		case token.TokenKind_Keyword_ABORT:
			return ConflictActionAbort
		case token.TokenKind_Keyword_FAIL:
			return ConflictActionFail
		case token.TokenKind_Keyword_IGNORE:
			return ConflictActionIgnore
		case token.TokenKind_Keyword_REPLACE:
			return ConflictActionReplace
		}
	}
	return ConflictActionNone
}

type unresolvedForeignKey struct {
	FromTable   *ast.CatalogObjectIdentifier
	FromColumns []ast.Identifier
	// ToTable is the table being referenced (the "parent" table).
	ToTable   *ast.CatalogObjectIdentifier
	ToColumns []ast.Identifier
}

type ForeignKey struct {
	AstNode interface {
		Accept(ast.Visitor)
	}

	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []ColumnLike

	Unresolved unresolvedForeignKey

	// ToTable is the table being referenced (the "parent" table).
	ToTable   *Table
	ToColumns []ColumnLike
}

func ForeignKeyFromColumnConstraintAst(colLike ColumnLike, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_ForeignKey) error {

	if constraints.FK != nil {
		// error foreign key already declared on column
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.FkClause.ReferencesKeyword.FileLoc).
			WithMessage("column has already delcared a foreign key constraint").
			WithLabels(
				report.LabelFromKeyword(constraints.FK.AstNode.(*ast.ColumnConstraint_ForeignKey).FkClause.ReferencesKeyword, "previous foreign key constraint declared here"),
			)
	}

	constraints.FK = &ForeignKey{
		AstNode: constraint,
		Unresolved: unresolvedForeignKey{
			FromColumns: []ast.Identifier{
				*colLike.GetName(),
			},

			ToTable:   &constraint.FkClause.ForeignTable,
			ToColumns: constraint.FkClause.ForeignColumns,
		},
	}

	return nil
}

func ForeignKeyFromTableConstraintAst(table *Table, constraint *ast.TableConstraint_ForeignKey) (*ForeignKey, error) {
	errs := []error{}

	result := &ForeignKey{
		AstNode:     constraint,
		FromTable:   table,
		FromColumns: make([]ColumnLike, len(constraint.Columns)),
		Unresolved: unresolvedForeignKey{
			FromTable:   table.CreateTable.TableIdentifier,
			FromColumns: constraint.Columns,
			ToTable:     &constraint.FkClause.ForeignTable,
			ToColumns:   make([]ast.Identifier, len(constraint.FkClause.ForeignColumns)),
		},
	}

	// resolve the local columns
	result.FromColumns = make([]ColumnLike, len(constraint.Columns))
	for i, fromCol := range constraint.Columns {
		col, ok := table.Columns[fromCol.Text]
		if !ok {
			// this is an symantic error, the local column does not exist
			err := report.
				NewReport("invalid foreign key definition").
				WithLocation(fromCol.FileLoc).
				WithLabels(
					report.LabelFromIdentifier(table.CreateTable.TableIdentifier.ObjectName, fmt.Sprintf("this table does not define column '%v'", fromCol.Text)),
					report.LabelFromIdentifier(fromCol, "this column does not exist"),
				)

			errs = append(errs, err)
			// @TODO(woody): this is a possible bug, we aren't added the failed column to the resolved columns
			// but there is a slot ready for it
			continue
		}

		// we have a column, so we can mark it as resolved
		result.FromColumns[i] = col
	}

	return result, errors.Join(errs...)
}

type Unique struct {
	Node interface {
		ast.Equalable
	}
	Name           *ast.Identifier
	IndexedColumns []ast.IndexedColumn
	ConflictAction ConflictAction
}

func (u *Unique) Eq(otherUnique any) bool {
	other, ok := otherUnique.(*Unique)
	if !ok {
		return false
	}

	return u.Node.Eq(other.Node)
}

func UniqueFromColumnConstraintAst(colLike ColumnLike, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_Unique) error {

	if constraints.Unique != nil {
		// unique is already declared error
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.UniqueKeyword.FileLoc).
			WithMessage("column has already delcared a unique constraint").
			WithLabels(
				report.LabelFromKeyword(constraints.Unique.Node.(*ast.ColumnConstraint_Unique).UniqueKeyword, "previous unique constraint declared here"),
			)
	}

	constraints.Unique = &Unique{
		Node: constraint,
		IndexedColumns: []ast.IndexedColumn{
			{Subject: colLike.GetName()},
		},
		ConflictAction: ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		constraints.Unique.Name = &constraint.Name.Name
	}

	return nil
}

func UniqueFromTableConstraintAst(table *Table, constraint *ast.TableConstraint_Unique) (*Unique, error) {
	result := &Unique{
		Node:           constraint,
		IndexedColumns: constraint.IndexedColumns,
		ConflictAction: ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		result.Name = &constraint.Name.Name
	}

	return result, nil
}

type Check struct {
	Node interface {
		ast.Equalable
	}
	Name *ast.Identifier
	Expr ast.Expr
}

func (check *Check) Eq(otherAny any) bool {
	other, ok := otherAny.(*Check)
	if !ok {
		return false
	}

	return check.Node.Eq(other.Node)
}

func CheckFromColumnConstraintAst(constraints *ColumnConstraints, constraint *ast.ColumnConstraint_Check) error {

	if constraints.Check != nil {
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.CheckKeyword.FileLoc).
			WithMessage("column has already delcared a check constraint").
			WithLabels(
				report.LabelFromExpr(constraints.Check.Expr, "previous check constraint declared here"),
			)
	}

	constraints.Check = &Check{
		Node: constraint,
		Expr: constraint.Expr,
	}

	if constraint.Name != nil {
		constraints.Check.Name = &constraint.Name.Name
	}

	return nil
}

func CheckFromTableConstraintAst(table *Table, constraint *ast.TableConstraint_Check) (*Check, error) {
	result := &Check{
		Node: constraint,
		Expr: constraint.Expr,
	}

	if constraint.Name != nil {
		result.Name = &constraint.Name.Name
	}

	// verify that the expr is semantically correct

	return result, nil
}

type CollateConstraint struct {
	Node interface {
		ast.Equalable
	}

	Name    *ast.Identifier
	Collate *ast.Identifier
}

func CollateConstraintFromColumnConstraintAst(constraints *ColumnConstraints, constraint *ast.ColumnConstraint_Collate) error {
	if constraints.Collate != nil {
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.Collate.CollateKeyword.FileLoc).
			WithMessage("column has already delcared a collate constraint").
			WithLabels(
				report.LabelFromIdentifier(*constraints.Collate.Collate, "previous collate constraint declared here"),
			)
	}

	constraints.Collate = &CollateConstraint{
		Collate: &constraint.Collate.Name,
	}

	if constraint.Name != nil {
		constraints.Collate.Name = &constraint.Name.Name
	}

	return nil
}

type NotNull struct {
	Node           *ast.ColumnConstraint_NotNull
	Name           *ast.Identifier
	ConflictClause ConflictAction
}

func NotNullFromColumnConstraint(constraints *ColumnConstraints, constraint *ast.ColumnConstraint_NotNull) error {

	if constraints.NotNull != nil {
		// error not null already declared
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.NotKeyword.FileLoc).
			WithMessage("column has already delcared a not null constraint").
			WithLabels(
				report.LabelFromKeyword(constraint.NotKeyword, "previous not null constraint declared here"),
			)
	}

	constraints.NotNull = &NotNull{
		Node:           constraint,
		ConflictClause: ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		constraints.NotNull.Name = &constraint.Name.Name
	}

	return nil
}

type Default struct {
	Node *ast.ColumnConstraint_Default

	Name *ast.Identifier
	Expr ast.Expr
}

func DefaultFromColumnConstraint(constraints *ColumnConstraints, constraint *ast.ColumnConstraint_Default) error {
	if constraints.Default != nil {
		// default constraint already declared error
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.DefaultKeyword.FileLoc).
			WithMessage("column has already delcared a default value").
			WithLabels(
				report.LabelFromKeyword(constraint.DefaultKeyword, "previous default constraint declared here"),
			)
	}

	constraints.Default = &Default{
		Expr: constraint.Default,
	}

	if constraint.Name != nil {
		constraints.Default.Name = &constraint.Name.Name
	}

	return nil
}
