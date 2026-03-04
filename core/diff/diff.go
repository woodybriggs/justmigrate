package diff

import (
	"errors"
	"fmt"
	"slices"
	"woodybriggs/justmigrate/core/ast"
)

type Diff struct{}

var (
	ErrArgumentMismatch error = errors.New("arguments a and b do not match")
)

func filterForCreateTable(value ast.Statement) (*ast.CreateTable, bool) {
	result, ok := value.(*ast.CreateTable)
	return result, ok
}

func isSameCreateTable(a, b *ast.CreateTable) bool {
	return a.TableIdentifier.Eq(b.TableIdentifier)
}

func resolveMissingTables(
	removed []*ast.CreateTable,
	added []*ast.CreateTable,
) (finalRemoved []*ast.CreateTable, finalAdded []*ast.CreateTable, ops []Op) {

	unresolvedRemovedTables := removed

	for _, newTable := range added {
		fmt.Printf("New table detected %s\n", newTable.TableIdentifier.ObjectName.Text)

		fmt.Println("this is a new table")
		for _, unresolved := range unresolvedRemovedTables {
			fmt.Printf("renamed from %s\n", unresolved.TableIdentifier.ObjectName.Text)
		}
	}

	return removed, finalAdded, ops
}

func (diff *Diff) DiffSchema(src, tgt []ast.Statement) ([]Op, error) {
	ops := []Op{}

	// Compare all create table statements
	{
		src := slices.Collect(filterThenMap(slices.Values(src), filterForCreateTable))
		tgt := slices.Collect(filterThenMap(slices.Values(tgt), filterForCreateTable))

		maybeRemovedTables, maybeAddedTables := symmetricDifference(src, tgt, isSameCreateTable)
		maybeModifiedTables := intersection(src, tgt, isSameCreateTable)

		removedTables, addedTables, renamedTableOps := resolveMissingTables(maybeRemovedTables, maybeAddedTables)

		for _, removedTable := range removedTables {
			ops = append(ops, &DelTableOp{removedTable.TableIdentifier})
		}

		for _, addedTable := range addedTables {
			ops = append(ops, &NewTableOp{addedTable})
		}

		ops = append(ops, renamedTableOps...)

		for _, pair := range maybeModifiedTables {
			tableOps := diff.DiffCreateTable(pair.A, pair.B)
			if tableOps != nil {
				ops = append(ops, tableOps...)
			}
		}
	}

	return ops, nil
}

func isSameColumnDefinition(a, b ast.ColumnDefinition) bool {
	return a.ColumnName.Eq(&b.ColumnName)
}

func isSameTableConstraint(a, b ast.TableConstraint) bool {
	switch a := a.(type) {
	case *ast.TableConstraint_PrimaryKey:
		_, ok := b.(*ast.TableConstraint_PrimaryKey)
		if !ok {
			return false
		}
		return true
	case *ast.TableConstraint_ForeignKey:
		b, ok := b.(*ast.TableConstraint_ForeignKey)
		if !ok {
			return false
		}
		return a.Eq(b)
	default:
		return false
	}
}

func (diff *Diff) DiffCreateTable(src, tgt *ast.CreateTable) []Op {
	ops := []Op{}

	// Compare column definitions
	{
		a := src.TableDefinition.ColumnDefinitions
		b := tgt.TableDefinition.ColumnDefinitions

		removedColumns, addedColumns := symmetricDifference(a, b, isSameColumnDefinition)
		maybeModifiedColumns := intersection(a, b, isSameColumnDefinition)

		for _, removedColumn := range removedColumns {
			ops = append(ops, &DelColOp{Table: src.TableIdentifier, Col: &removedColumn.ColumnName})
		}

		for _, addedColumn := range addedColumns {
			ops = append(ops, &NewColOp{Table: src.TableIdentifier, Col: &addedColumn})
		}

		for _, pair := range maybeModifiedColumns {
			columnOps := diff.DiffColumnDefinition(src.TableIdentifier, pair.A, pair.B)
			if columnOps != nil {
				ops = append(ops, columnOps...)
			}
		}
	}

	/*
		We need to extract out the table constraints by type.

		1. We know that there is only ever one PK constraint per table
		2. We can collect and compare all FK constraints
		3. We can collect and compare all UNIQUE constraints
		4. We can collect and compare all CHECK constraints
	*/

	return ops
}

func (diff *Diff) DiffColumnDefinition(table *ast.CatalogObjectIdentifier, src, tgt ast.ColumnDefinition) []Op {
	ops := []Op{}

	if !src.TypeName.Eq(tgt.TypeName) {
		ops = append(ops, &ChangeColTypeOp{Table: table, Col: &src.ColumnName, TypeName: tgt.TypeName})
	}

	/*
		We need to pick out column constraints that are 1 per col
		seperate them from CHECK constraints as 1 column can have many check constraints.

		should we lift column constraints that can be defined as table constraints up to the table?
		if we did we would be closer to our goal of semantic analysis, moving away from syntax more.

		1. Primary Key constraint
		2. Foreign Key (REFERENCES) constraint
		3. Default constraint
		4. Not Null constraint
		5. Unique constraint
		6. Collate constraint
		7. As constraint
	*/

	return ops
}

func (diff *Diff) DiffTableConstraint(table *ast.CatalogObjectIdentifier, a, b ast.TableConstraint) []Op {
	ops := []Op{}

	// do something here

	return ops
}
