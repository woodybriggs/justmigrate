package diff

import (
	"errors"
	"fmt"
	"slices"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/prompt"
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

func resolveMissingColumns(
	table *ast.CatalogObjectIdentifier,
	removed []ast.ColumnDefinition,
	added []ast.ColumnDefinition,
) (finalRemoved []ast.ColumnDefinition, finalAdded []ast.ColumnDefinition, ops []Op) {
	if len(removed) == 0 {
		return removed, added, nil
	}

	unresolvedRemovedCols := removed
	terminal := prompt.Terminal{}
	terminal.Start()
	defer terminal.Restore()

	for _, newCol := range added {

		// if we have resolved all the "removed" columns,
		// then it can be assumed that any column that
		// hasn't been marked as renamed is definatly a new column
		if len(unresolvedRemovedCols) == 0 {
			finalAdded = append(finalAdded, newCol)
			continue
		}

		options := []prompt.SelectOption{
			{
				Label: fmt.Sprintf("new column: %s", newCol.ColumnName.Text),
				Value: &NewColOp{Table: table, Col: &newCol},
			},
		}

		for _, unresolvedCol := range unresolvedRemovedCols {
			options = append(options, prompt.SelectOption{
				Label: fmt.Sprintf("renamed from: %s", unresolvedCol.ColumnName.Text),
				Value: &RenameColOp{Table: table, FromCol: &unresolvedCol.ColumnName, ToCol: &newCol.ColumnName},
			})
		}

		sel := prompt.Select{}
		title := fmt.Sprintf("Resolve table %s: Is this column new or renamed?", newCol.ColumnName.Text)
		choiceIndex, err := sel.Do(&terminal, title, options)
		if err != nil {
			panic(err)
		}
		op := options[choiceIndex]
		switch typ := op.Value.(type) {
		case *RenameColOp:
			// remove the From column from the unresolved columns as it is now resolved
			unresolvedRemovedCols = slices.DeleteFunc(unresolvedRemovedCols, func(col ast.ColumnDefinition) bool {
				return col.ColumnName.Eq(typ.FromCol)
			})

			// add the rename op to the output
			ops = append(ops, typ)
		case *NewColOp:
			// add the new column to final added
			finalAdded = append(finalAdded, *typ.Col)
		}
	}
	// any unresolved removed columns are now final as removed
	finalRemoved = unresolvedRemovedCols
	return
}

func resolveMissingTables(
	removed []*ast.CreateTable,
	added []*ast.CreateTable,
) (finalRemoved []*ast.CreateTable, finalAdded []*ast.CreateTable, ops []Op) {

	if len(removed) == 0 {
		return removed, added, nil
	}

	unresolvedRemovedTables := removed
	terminal := prompt.Terminal{}
	terminal.Start()
	defer terminal.Restore()

	for _, newTable := range added {

		// if we have resolved all the "removed" columns,
		// then it can be assumed that any column that
		// hasn't been marked as renamed is definatly a new column
		if len(unresolvedRemovedTables) == 0 {
			finalAdded = append(finalAdded, newTable)
			continue
		}

		options := []prompt.SelectOption{
			{
				Label: fmt.Sprintf("new table: %s", newTable.TableIdentifier.ObjectName.Text),
				Value: &NewTableOp{newTable},
			},
		}

		for _, unresolved := range unresolvedRemovedTables {
			options = append(options, prompt.SelectOption{
				Label: fmt.Sprintf("renamed from:  %s", unresolved.TableIdentifier.ObjectName.Text),
				Value: &RenameTableOp{From: unresolved.TableIdentifier, To: newTable.TableIdentifier},
			})
		}

		sel := prompt.Select{}
		title := fmt.Sprintf("Resolve table %s: Is this table new or renamed?", newTable.TableIdentifier.ObjectName.Text)
		choiceIndex, err := sel.Do(&terminal, title, options)
		if err != nil {
			panic(err)
		}
		op := options[choiceIndex]
		switch typ := op.Value.(type) {
		case *RenameTableOp:
			// remove the From table from the unresolved table as it is now resolved
			unresolvedRemovedTables = slices.DeleteFunc(unresolvedRemovedTables, func(table *ast.CreateTable) bool {
				return table.TableIdentifier.Eq(typ.From)
			})

			// add the rename op to the output
			ops = append(ops, typ)
		case *NewTableOp:
			// add the new table to final added
			finalAdded = append(finalAdded, typ.CreateTable)
		}
	}

	// any unresolved removed tables are now final as removed
	finalRemoved = unresolvedRemovedTables
	return
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

		maybeRemovedColumns, maybeAddedColumns := symmetricDifference(a, b, isSameColumnDefinition)
		maybeModifiedColumns := intersection(a, b, isSameColumnDefinition)

		removedColumns, addedColumns, renamedColumnsOps := resolveMissingColumns(src.TableIdentifier, maybeRemovedColumns, maybeAddedColumns)

		for _, removedColumn := range removedColumns {
			ops = append(ops, &DelColOp{Table: src.TableIdentifier, Col: &removedColumn.ColumnName})
		}

		for _, addedColumn := range addedColumns {
			ops = append(ops, &NewColOp{Table: src.TableIdentifier, Col: &addedColumn})
		}

		ops = append(ops, renamedColumnsOps...)

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
