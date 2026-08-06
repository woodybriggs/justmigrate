package diff

import (
	"errors"
	"fmt"
	"justmigrate/internal/backend/op"
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/prompt"
	"maps"
	"runtime"
	"slices"
)

var (
	ErrArgumentMismatch error = errors.New("arguments a and b do not match")
)

func resolveMissingColumns(
	schem *schema.Schema,
	table *schema.Table,
	removed []schema.ColumnLike,
	added []schema.ColumnLike,
) (finalRemoved []schema.ColumnLike, finalAdded []schema.ColumnLike, ops []op.Op) {
	if len(removed) == 0 {
		return removed, added, nil
	}

	unresolvedRemovedCols := removed
	terminal := prompt.Terminal{}
	err := terminal.Start()
	if err != nil {
		return removed, added, ops
	}
	defer terminal.Restore()

	for _, newCol := range added {
		// if we have resolved all the "removed" columns,
		// then it can be assumed that any column that
		// hasn't been marked as renamed is for sure a new column
		if len(unresolvedRemovedCols) == 0 {
			finalAdded = append(finalAdded, newCol)
			continue
		}

		// prepare the options
		options := []prompt.SelectOption{
			{
				Label: fmt.Sprintf("new column: %s", newCol.GetName()),
				Value: &op.AddColumn{
					Target: op.TargetTable{
						Schema: schem,
						Table:  table,
					},
					Column: newCol,
				},
			},
		}
		for _, unresolvedCol := range unresolvedRemovedCols {
			options = append(options, prompt.SelectOption{
				Label: fmt.Sprintf("renamed from: %s", unresolvedCol.GetName()),
				Value: &op.RenameColumn{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: unresolvedCol,
					},
					NewName: newCol.GetName(),
				},
			})
		}

		sel := prompt.Select{}
		title := fmt.Sprintf("Resolve column %s.%s: Is this column new or renamed?", table.Name, newCol.GetName())
		choiceIndex, err := sel.Do(&terminal, title, options)
		if err != nil {
			panic(err)
		}
		operation := options[choiceIndex]
		switch operation := operation.Value.(type) {
		case *op.RenameColumn:
			// remove the From column from the unresolved columns as it is now resolved
			unresolvedRemovedCols = slices.DeleteFunc(unresolvedRemovedCols, func(col schema.ColumnLike) bool {
				return col.GetName().Eq(operation.NewName)
			})

			// add the rename op to the output
			ops = append(ops, operation)
		case *op.AddColumn:
			// add the new column to final added
			finalAdded = append(finalAdded, operation.Column)
		}
	}
	// any unresolved removed columns are now final as removed
	finalRemoved = unresolvedRemovedCols
	return
}

func resolveMissingTables(
	schem *schema.Schema,
	removed []*schema.Table,
	added []*schema.Table,
) (finalRemoved []*schema.Table, finalAdded []*schema.Table, ops []op.Op) {

	if len(removed) == 0 {
		return removed, added, nil
	}

	unresolvedRemovedTables := removed
	terminal := prompt.Terminal{}
	if err := terminal.Start(); err != nil {
		return removed, added, ops
	}
	defer terminal.Restore()

	for _, newTable := range added {

		// if we have resolved all the "removed" tables,
		// then it can be assumed that any table that
		// hasn't been marked as renamed is for sure a new table
		if len(unresolvedRemovedTables) == 0 {
			finalAdded = append(finalAdded, newTable)
			continue
		}

		// prepare the options
		options := []prompt.SelectOption{
			{
				Label: fmt.Sprintf("new table: %s", newTable.Name),
				Value: &op.AddTable{
					Target: op.TargetSchema{
						Schema: schem,
					},
					Table: newTable,
				},
			},
		}

		for _, unresolved := range unresolvedRemovedTables {
			options = append(options, prompt.SelectOption{
				Label: fmt.Sprintf("renamed from:  %s", unresolved.Name),
				Value: &op.RenameTable{
					Target: op.TargetTable{
						Schema: schem,
						Table:  unresolved,
					},
					NewName: newTable.Node.TableIdentifier,
				},
			})
		}

		sel := prompt.Select{}
		title := fmt.Sprintf("Resolve table %s: Is this table new or renamed?", newTable.Name)
		choiceIndex, err := sel.Do(&terminal, title, options)
		if err != nil {
			panic(err)
		}
		operation := options[choiceIndex]
		switch operation := operation.Value.(type) {
		case *op.RenameTable:
			// remove the From table from the unresolved table as it is now resolved
			unresolvedRemovedTables = slices.DeleteFunc(unresolvedRemovedTables, func(table *schema.Table) bool {
				return table.Node.TableIdentifier.Eq(operation.NewName)
			})
			// add the rename op to the output
			ops = append(ops, operation)
		case *op.AddTable:
			// add the new table to final added
			finalAdded = append(finalAdded, operation.Table)
		}
	}

	// any unresolved removed tables are now final as removed
	finalRemoved = unresolvedRemovedTables
	return
}

func diffState[T any](a, b *T, ifA, ifB func(*T), cmp func(a, b *T)) {
	if a != nil && b == nil {
		ifA(a)
	} else if a == nil && b != nil {
		ifB(b)
	} else if a != nil && b != nil {
		cmp(a, b)
	}
}

func DiffSchema(src, tgt *schema.Schema) (ops []op.Op, err error) {

	// Compare all create table statements
	{
		isSameTable := func(a *schema.Table, b *schema.Table) bool {
			return a.Name == b.Name
		}

		a := slices.Collect(maps.Values(src.Tables))
		b := slices.Collect(maps.Values(tgt.Tables))

		maybeRemoved, maybeAdded := symmetricDifferenceFunc(a, b, isSameTable)
		maybeModified := intersectionFunc(a, b, isSameTable)

		removedTables, addedTables, renamedTableOps := resolveMissingTables(src, maybeRemoved, maybeAdded)

		for _, removedTable := range removedTables {
			ops = append(ops, &op.DropTable{
				Target: op.TargetTable{
					Schema: src,
					Table:  removedTable,
				},
			})
		}

		for _, addedTable := range addedTables {
			ops = append(ops, &op.AddTable{
				Target: op.TargetSchema{
					Schema: src,
				},
				Table: addedTable,
			})
		}

		ops = append(ops, renamedTableOps...)

		for _, pair := range maybeModified {
			tableOps := DiffCreateTable(src, pair.A, pair.B)
			if tableOps != nil {
				ops = append(ops, tableOps...)
			}
		}
	}

	return ops, nil
}

func DiffCreateTable(schem *schema.Schema, src, tgt *schema.Table) []op.Op {
	ops := []op.Op{}

	{
		isSameColumn := func(a, b schema.ColumnLike) bool {
			return a.GetName().Eq(b.GetName())
		}

		// Compare column definitions
		a := slices.Collect(maps.Values(src.Columns))
		b := slices.Collect(maps.Values(tgt.Columns))

		maybeRemovedColumns, maybeAddedColumns := symmetricDifferenceFunc(a, b, isSameColumn)
		maybeModifiedColumns := intersectionFunc(a, b, isSameColumn)

		removedColumns, addedColumns, renamedColumnsOps := resolveMissingColumns(schem, src, maybeRemovedColumns, maybeAddedColumns)

		for _, removedColumn := range removedColumns {
			ops = append(ops, &op.DropColumn{
				Target: op.TargetColumn{
					Schema: schem,
					Table:  src,
					Column: removedColumn,
				},
			})
		}

		for _, addedColumn := range addedColumns {
			ops = append(ops, &op.AddColumn{
				Target: op.TargetTable{
					Schema: schem,
					Table:  src,
				},
				Column: addedColumn,
			})
		}

		ops = append(ops, renamedColumnsOps...)
		for _, pair := range maybeModifiedColumns {
			columnOps := DiffColumnDefinition(schem, src, pair.A, pair.B)
			if columnOps != nil {
				ops = append(ops, columnOps...)
			}
		}
	}

	diffState(
		src.Constraints.PK, tgt.Constraints.PK,
		func(pk *schema.PrimaryKey) {
			ops = append(ops, op.MakeDropConstraintFromPrimaryKey(schem, src, pk))
		},
		func(pk *schema.PrimaryKey) {
			ops = append(ops, op.MakeAddConstraintFromPrimaryKey(schem, src, pk))
		},
		func(a, b *schema.PrimaryKey) {
			// both tables have a primary key, we need to check if it has changed
			if a.Eq(b) {
				return // nothing to do, both primary keys are the same
			}

			// primary key has changed, we can reconcile this with a drop constraint + add constraint for now
			ops = append(ops,
				op.MakeDropConstraintFromPrimaryKey(schem, src, a),
				op.MakeAddConstraintFromPrimaryKey(schem, src, b),
			)
		},
	)

	// @TODO(woody): need to do foreign keys
	{
		a := src.Constraints.FKs
		b := tgt.Constraints.FKs

		maybeRemoved, maybeAdded := symmetricDifference(a, b)
		maybeModified := intersection(maybeRemoved, maybeAdded)
		_ = maybeModified
	}

	{
		// check for changes in CHECK constraints
		// this checks only for added and removed
		// as there is no effect of modifiying a CHECK constraint
		a := src.Constraints.Checks
		b := tgt.Constraints.Checks

		removed, added := symmetricDifference(a, b)

		for _, r := range removed {
			switch r.Node.(type) {
			case *ast.ColumnConstraint_Check:
				// skip because this would be handled when diffing the column
			case *ast.TableConstraint_Check:
				ops = append(ops, &op.DropTableConstraint{
					Target: op.TargetTableConstraint{
						Schema:     schem,
						Table:      src,
						Constraint: r.Node.(ast.TableConstraint),
					},
				})
			}
		}

		for _, a := range added {
			switch a.Node.(type) {
			case *ast.ColumnConstraint_Check:
				// skip because this would be handled when diffing the column
			case *ast.TableConstraint_Check:
				ops = append(ops, &op.AddTableConstraint{
					Target: op.TargetTable{
						Schema: schem,
						Table:  src,
					},
					Constraint: a.Node.(ast.TableConstraint),
				})
			}
		}
	}

	{
		a := src.Constraints.Uniques
		b := src.Constraints.Uniques

		removed, added := symmetricDifference(a, b)

		for _, r := range removed {
			ops = append(ops, &op.DropTableConstraint{
				Target: op.TargetTableConstraint{
					Schema:     schem,
					Table:      src,
					Constraint: r.Node.(ast.TableConstraint),
				},
			})
		}

		for _, a := range added {
			ops = append(ops, &op.AddTableConstraint{
				Target: op.TargetTable{
					Schema: schem,
					Table:  src,
				},
				Constraint: a.Node.(ast.TableConstraint),
			})
		}
	}

	return ops
}

func __func__() string {
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	return fmt.Sprint(fn.Name(), file, line)
}

func DiffColumnDefinition(schem *schema.Schema, table *schema.Table, src, tgt schema.ColumnLike) []op.Op {
	ops := []op.Op{}

	switch src := src.(type) {
	case *schema.Column:
		{
			if _, ok := tgt.(*schema.Column); ok {
				// if they are the same type we can break out and continue diffing
				break
			}
			// if the tgt col is a generated column
			// we need to drop the column, and create the generated column
			// this would be destructive and cause data loss
			_ = src
			return ops
		}
	case *schema.GeneratedColumn:
		{
			if _, ok := tgt.(*schema.GeneratedColumn); ok {
				// if they are the same type we can break out and continue diffing
				break
			}
			// if the tgt col is a regular column
			// we need to drop the generated column, create the regular column
			// this would be destructive and cause data loss
			// could recommend to user to
			// 1. create new column
			// 2. copy over data from generated
			// 3. delete generated column
			_ = src

			return ops
		}
	}

	diffState(
		src.GetType(), tgt.GetType(),
		func(a *schema.Type) {

			panic("@TODO(woody): type has been dropped")
		},
		func(b *schema.Type) {
			panic("@TODO(woody): type has been added")
		},
		func(a, b *schema.Type) {
			if a.Eq(b) {
				return
			}
			panic("@TODO(woody): check if type has changed")
		},
	)

	{
		a := src.GetConstraints()
		b := tgt.GetConstraints()

		diffState(
			a.NotNull, b.NotNull,
			func(a *schema.NotNull) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node,
					},
				})
			},
			func(b *schema.NotNull) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node,
				})
			},
			func(a, b *schema.NotNull) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if conflict clause has changed")
			},
		)

		diffState(
			a.Default, b.Default,
			func(a *schema.Default) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node,
					},
				})
			},
			func(b *schema.Default) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node,
				})
			},
			func(a, b *schema.Default) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if expression has changed")
			},
		)

		diffState(
			a.Unique, b.Unique,
			func(a *schema.Unique) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node.(*ast.ColumnConstraint_Unique),
					},
				})
			},
			func(b *schema.Unique) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node.(*ast.ColumnConstraint_Unique),
				})
			},
			func(a, b *schema.Unique) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if indexed columns have changed, or conflict clause")
			},
		)

		diffState(
			a.FK, b.FK,
			func(a *schema.ForeignKey) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node.(*ast.ColumnConstraint_ForeignKey),
					},
				})
			},
			func(b *schema.ForeignKey) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node.(*ast.ColumnConstraint_ForeignKey),
				})
			},
			func(a, b *schema.ForeignKey) {
				if a.Eq(b) {
					return
				}

				panic("@TODO(woody): check if foreign key has changed")
			},
		)

		diffState(
			a.PK, b.PK,
			func(a *schema.PrimaryKey) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node.(*ast.ColumnConstraint_PrimaryKey),
					},
				})
			},
			func(b *schema.PrimaryKey) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node.(*ast.ColumnConstraint_PrimaryKey),
				})
			},
			func(a, b *schema.PrimaryKey) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if primary key has changed")
			},
		)

		diffState(
			a.Check, b.Check,
			func(a *schema.Check) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node.(*ast.ColumnConstraint_Check),
					},
				})
			},
			func(b *schema.Check) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node.(*ast.ColumnConstraint_Check),
				})
			},
			func(a, b *schema.Check) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if expr has changed")
			},
		)

		diffState(
			a.Collate, b.Collate,
			func(a *schema.CollateConstraint) {
				ops = append(ops, &op.DropColumnConstraint{
					Target: op.TargetColumnConstraint{
						Schema:     schem,
						Table:      table,
						Column:     src,
						Constraint: a.Node.(*ast.ColumnConstraint_Collate),
					},
				})
			},
			func(b *schema.CollateConstraint) {
				ops = append(ops, &op.AddColumnConstraint{
					Target: op.TargetColumn{
						Schema: schem,
						Table:  table,
						Column: src,
					},
					Constraint: b.Node.(*ast.ColumnConstraint_Collate),
				})
			},
			func(a, b *schema.CollateConstraint) {
				if a.Eq(b) {
					return
				}
				panic("@TODO(woody): check if collation has changed")
			},
		)
	}

	return ops
}

func DiffTableConstraint(table *ast.CatalogObjectIdentifier, a, b ast.TableConstraint) []op.Op {
	ops := []op.Op{}

	// i think we can only check for named constraints here

	return ops
}
