package schema

import (
	"errors"
	"fmt"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/report"
	"justmigrate/internal/frontend/token"
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

func (pk *PrimaryKey) Eq(otherAny any) bool {
	other, ok := otherAny.(*PrimaryKey)
	if !ok {
		return false
	}

	return pk.Node.Eq(other.Node)
}

func PrimaryKeyFromTableConstraintAst(
	table *Table,
	constraints *TableConstraints,
	constraint *ast.TableConstraint_PrimaryKey,
) error {

	if constraints.PK != nil {
		return report.
			NewReport("primary key already defined").
			WithLocation(constraint.PrimaryKeyword.FileLoc).
			WithLabels(
				report.LabelFromIdentifier(*constraints.PK.Name, "primary key already defined here"),
			)
	}

	errs := []error{}
	pk := &PrimaryKey{
		Node:            constraint,
		IndexedColumns:  constraint.IndexedColumns,
		ResolvedColumns: make([]ColumnLike, len(constraint.IndexedColumns)),
		ConflictClause:  ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		pk.Name = &constraint.Name.Name
	}

	// for each indexed column we check that it is a valid column
	for _, indexedCol := range constraint.IndexedColumns {
		ident, ok := indexedCol.Subject.(*ast.Identifier)
		if !ok {
			errs = append(errs, report.
				NewReport("semmantic error").
				WithLocation(report.LocationFromExpr(indexedCol.Subject)).
				WithMessage("expressions not supported in primary key indexed columns").
				WithLabels(
					report.LabelFromExpr(indexedCol.Subject, "this must be a column name within the table"),
				),
			)
			continue
		}

		col, ok := table.Columns[ident.String()]
		if !ok {
			errs = append(errs, report.
				NewReport("semmantic error").
				WithLocation(report.LocationFromExpr(indexedCol.Subject)).
				WithMessage("column does not exist in table").
				WithLabels(
					report.LabelFromIdentifier(table.Node.TableIdentifier.ObjectName, "this table"),
					report.LabelFromExpr(indexedCol.Subject, "does not declare this column"),
				),
			)
			continue
		}

		if _, ok := col.(*GeneratedColumn); ok {
			errs = append(errs, report.
				NewReport("semmantic error").
				WithLocation(report.LocationFromExpr(indexedCol.Subject)).
				WithMessage("can not use generated columns in primary key constraint").
				WithLabels(
					report.LabelFromIdentifier(*col.GetName(), "this column"),
					report.LabelFromExpr(indexedCol.Subject, "can not be used here"),
				),
			)
			continue
		}
	}

	joinedErrs := errors.Join(errs...)
	if joinedErrs == nil {
		constraints.PK = pk
	}

	return joinedErrs
}

func PrimaryKeyFromColumnConstraintAst(column ColumnLike, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_PrimaryKey) error {

	if constraints.PK != nil {
		// primary key already declared on column error
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.PrimaryKeyword.FileLoc).
			WithMessage("column has already delcared a foreign key constraint").
			WithLabels(
				report.LabelFromKeyword(constraints.PK.Node.(*ast.ColumnConstraint_PrimaryKey).PrimaryKeyword, "first primary key constraint declared here"),
				report.LabelFromKeyword(constraint.PrimaryKeyword, "second primary key constraint declared here"),
			)
	}

	localPK := &PrimaryKey{
		Node: constraint,
		IndexedColumns: []ast.IndexedColumn{
			{
				Subject: column.GetName(),
				Order:   constraint.Order,
			},
		},
		ResolvedColumns: []ColumnLike{column},
		ConflictClause:  ConflictClauseFromAst(constraint.ConflictClause),
	}

	if constraint.Name != nil {
		localPK.Name = &constraint.Name.Name
	}

	constraints.PK = localPK

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
	Node interface {
		Accept(ast.Visitor)
		Equalable
	}

	Name *ast.Identifier

	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []ColumnLike

	Unresolved unresolvedForeignKey

	// ToTable is the table being referenced (the "parent" table).
	ToTable   *Table
	ToColumns []ColumnLike
}

func (fk *ForeignKey) Eq(otherAny any) bool {
	other, ok := otherAny.(*ForeignKey)
	if !ok {
		return false
	}

	return fk.Node.Eq(other.Node)
}

func ForeignKeyFromColumnConstraintAst(colName *ast.Identifier, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_ForeignKey) error {

	if constraints.FK != nil {
		// error foreign key already declared on column
		return report.NewReport("duplicate column constraint").
			WithLocation(constraint.FkClause.ReferencesKeyword.FileLoc).
			WithMessage("column has already delcared a foreign key constraint").
			WithLabels(
				report.LabelFromKeyword(constraints.FK.Node.(*ast.ColumnConstraint_ForeignKey).FkClause.ReferencesKeyword, "previous foreign key constraint declared here"),
			)
	}

	constraints.FK = &ForeignKey{
		Node: constraint,
		Unresolved: unresolvedForeignKey{
			FromColumns: []ast.Identifier{*colName},
			ToTable:     &constraint.FkClause.ForeignTable,
			ToColumns:   constraint.FkClause.ForeignColumns,
		},
	}

	if constraint.Name != nil {
		constraints.FK.Name = &constraint.Name.Name
	}

	return nil
}

func ForeignKeyFromTableConstraintAst(table *Table, constraint *ast.TableConstraint_ForeignKey) (*ForeignKey, error) {
	errs := []error{}

	result := &ForeignKey{
		Node:        constraint,
		FromTable:   table,
		FromColumns: make([]ColumnLike, len(constraint.Columns)),
		Unresolved: unresolvedForeignKey{
			FromTable:   table.Node.TableIdentifier,
			FromColumns: constraint.Columns,
			ToTable:     &constraint.FkClause.ForeignTable,
			ToColumns:   constraint.FkClause.ForeignColumns,
		},
	}

	if constraint.Name != nil {
		result.Name = &constraint.Name.Name
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
					report.LabelFromIdentifier(table.Node.TableIdentifier.ObjectName, fmt.Sprintf("this table does not define column '%v'", fromCol.Text)),
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

func ForeignKeyClauseAstFromForeignKey(fk *ForeignKey) *ast.ForeignKeyClause {

	if true {
		panic("@TODO(woody): we need to convert resolved columns to []ast.Identifier")
	}

	return ast.MakeForeignKeyClause(
		ast.Keyword(token.Keyword("REFERENCES")),
		*ast.MakeCatalogObjectIdentifier(
			nil,
			// @TODO(woody): we probably need to store schema name inside table
			*ast.MakeIdentifier(token.Token{Text: fk.ToTable.Name, Kind: token.TokenKind_Identifier}),
		),
		token.Token{Text: "(", Kind: token.TokenKind_LParen},
		fk.Unresolved.ToColumns,
		token.Token{Text: ")", Kind: token.TokenKind_RParen},
		// @TODO(woody) we dont store any of this in the foreign key at the moment
		nil,
		nil,
		nil,
	)
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

func UniqueFromColumnConstraintAst(colName *ast.Identifier, constraints *ColumnConstraints, constraint *ast.ColumnConstraint_Unique) error {

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
			{Subject: colName},
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

func (cc *CollateConstraint) Eq(otherAny any) bool {
	other, ok := otherAny.(*CollateConstraint)
	if !ok {
		return false
	}

	return cc.Node.Eq(other.Node)
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

func (nn *NotNull) Eq(otherAny any) bool {
	other, ok := otherAny.(*NotNull)
	if !ok {
		return false
	}

	return nn.ConflictClause == other.ConflictClause
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

func (def *Default) Eq(otherAny any) bool {
	other, ok := otherAny.(*Default)
	if !ok {
		return false
	}

	return other.Node.Eq(other.Node)
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
