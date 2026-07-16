package schema

import (
	"errors"
	"fmt"
	"slices"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
)

type TableConstraints struct {
	PK      *PrimaryKey
	FKs     []*ForeignKey
	Checks  []*Check
	Uniques []*Unique
}

type Table struct {
	CreateTable *ast.CreateTable
	Name        string
	Columns     map[string]ColumnLike
	Constraints TableConstraints
	Indexes     []*Index
}

func (t *Table) Eq(otherTable any) bool {
	other, ok := otherTable.(*Table)
	if !ok {
		return false
	}
	return t.CreateTable.Eq(other.CreateTable)
}

func TableFromAst(createTable *ast.CreateTable) (*Table, error) {
	// this is where we validate that the table is "correct" by itself

	// 1. normalize constraints at the column level up to the table constraints
	// 2. check that table constraint primary keys actually index columns that exist in the table
	// 3. check that foreign key columns actually exist in the table (no need to check for foreign column existance)
	// 4. check that table constraint check exprs that depend on table column, are real cols

	errs := []error{}

	table := &Table{
		CreateTable: createTable,
		Name:        createTable.TableIdentifier.String(),
		Columns:     map[string]ColumnLike{},
		Constraints: TableConstraints{},
	}

	// add all the columns
	for i := range createTable.TableDefinition.ColumnDefinitions {

		colDef := &createTable.TableDefinition.ColumnDefinitions[i]

		// determine if this column is generated or a regular column
		// we do this here because we populate the table as we add columns
		// we do this so that when we verify PrimaryKey's we can safely lookup a column by name
		// we do this so that a PrimaryKey schema object knows where it came from:
		// a TableConstraint or a ColumnConstraint
		isGenerated := slices.IndexFunc(colDef.ColumnConstraints, func(con ast.ColumnConstraint) bool {
			_, ok := con.(*ast.ColumnConstraint_Generated)
			return ok
		})

		var colLike ColumnLike = nil
		if isGenerated >= 0 {
			column := &GeneratedColumn{}
			err := GeneratedColumnFromAst(table, column, colDef)
			if err != nil {
				if colErrs, ok := err.(interface{ Unwrap() []error }); ok {
					errs = append(errs, colErrs.Unwrap()...)
				} else {
					errs = append(errs, err)
				}
				continue
			}
			colLike = column
		} else {
			column := &Column{}
			err := ColumnFromAst(table, column, &createTable.TableDefinition.ColumnDefinitions[i])
			if err != nil {
				if colErrs, ok := err.(interface{ Unwrap() []error }); ok {
					errs = append(errs, colErrs.Unwrap()...)
				} else {
					errs = append(errs, err)
				}
				continue
			}
			colLike = column
		}

		table.Columns[colLike.GetName().String()] = colLike

		constraints := colLike.GetConstraints()

		if constraints.PK != nil {
			table.Constraints.PK = constraints.PK
		}

		if constraints.FK != nil {
			table.Constraints.FKs = append(table.Constraints.FKs, constraints.FK)
		}

		if constraints.Check != nil {
			table.Constraints.Checks = append(table.Constraints.Checks, constraints.Check)
		}

		if constraints.Unique != nil {
			table.Constraints.Uniques = append(table.Constraints.Uniques, constraints.Unique)
		}
	}

	// now that the table has been fully defined, we need to resolve the internal wiring of fk
	for _, fk := range table.Constraints.FKs {
		// column defined fks need to be updated now that the table is fully defined
		if fk.FromTable == nil {
			fk.FromTable = table
		}

		fk.FromColumns = make([]ColumnLike, len(fk.Unresolved.FromColumns))
		for i, fromCol := range fk.Unresolved.FromColumns {
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
			fk.FromColumns[i] = col
		}
	}

	// add all the constraints
	for i := range createTable.TableDefinition.TableConstraints {
		switch constraint := createTable.TableDefinition.TableConstraints[i].(type) {
		case *ast.TableConstraint_PrimaryKey:
			{
				if table.Constraints.PK != nil {
					err := report.
						NewReport("primary key already defined").
						WithLocation(constraint.PrimaryKeyword.FileLoc).
						WithLabels(
							report.LabelFromIdentifier(*table.Constraints.PK.Name, "primary key already defined here"),
						)
					errs = append(errs, err)
					continue
				}

				primaryKey, err := PrimaryKeyFromTableConstraintAst(table, constraint)
				if err != nil {
					if pkErrs, ok := err.(interface{ Unwrap() []error }); ok {
						errs = append(errs, pkErrs.Unwrap()...)
					} else {
						errs = append(errs, err)
					}
					continue
				}

				table.Constraints.PK = primaryKey
			}
		case *ast.TableConstraint_ForeignKey:
			{
				foreignKey, err := ForeignKeyFromTableConstraintAst(table, constraint)
				if err != nil {
					if fkErrs, ok := err.(interface{ Unwrap() []error }); ok {
						errs = append(errs, fkErrs.Unwrap()...)
					} else {
						errs = append(errs, err)
					}
					continue
				}

				table.Constraints.FKs = append(table.Constraints.FKs, foreignKey)
			}
		case *ast.TableConstraint_Check:
			{
				check, err := CheckFromTableConstraintAst(table, constraint)
				if err != nil {
					if ckErrs, ok := err.(interface{ Unwrap() []error }); ok {
						errs = append(errs, ckErrs.Unwrap()...)
					} else {
						errs = append(errs, err)
					}
					continue
				}

				table.Constraints.Checks = append(table.Constraints.Checks, check)
			}
		case *ast.TableConstraint_Unique:
			{
				unique, err := UniqueFromTableConstraintAst(table, constraint)
				if err != nil {
					if fkErrs, ok := err.(interface{ Unwrap() []error }); ok {
						errs = append(errs, fkErrs.Unwrap()...)
					} else {
						errs = append(errs, err)
					}
					continue
				}

				table.Constraints.Uniques = append(table.Constraints.Uniques, unique)
			}
		}

	}

	return table, errors.Join(errs...)
}

func validateCheckExpr(table *Table, expr ast.Expr) error {
	return nil
}
