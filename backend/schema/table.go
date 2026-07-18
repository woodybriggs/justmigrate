package schema

import (
	"errors"
	"fmt"
	"slices"
	"woodybriggs/justmigrate/errext"
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
	Node        *ast.CreateTable
	Name        string
	Temporary   bool
	Columns     map[string]ColumnLike
	Constraints TableConstraints
	Indexes     []*Index
}

func (t *Table) Eq(otherTable any) bool {
	other, ok := otherTable.(*Table)
	if !ok {
		return false
	}
	return t.Node.Eq(other.Node)
}

func (table *Table) AddConstraint(constraint ast.TableConstraint) error {
	errs := []error{}
	switch constraint := constraint.(type) {
	case *ast.TableConstraint_PrimaryKey:
		{
			err := PrimaryKeyFromTableConstraintAst(table, &table.Constraints, constraint)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			}
		}
	case *ast.TableConstraint_ForeignKey:
		{
			foreignKey, err := ForeignKeyFromTableConstraintAst(table, constraint)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			} else {
				table.Constraints.FKs = append(table.Constraints.FKs, foreignKey)
			}
		}
	case *ast.TableConstraint_Check:
		{
			check, err := CheckFromTableConstraintAst(table, constraint)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			} else {
				table.Constraints.Checks = append(table.Constraints.Checks, check)
			}
		}
	case *ast.TableConstraint_Unique:
		{
			unique, err := UniqueFromTableConstraintAst(table, constraint)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			} else {
				table.Constraints.Uniques = append(table.Constraints.Uniques, unique)
			}
		}
	}

	return errors.Join(errs...)
}

func (table *Table) DropConstraint(constraint ast.TableConstraint) error {
	switch constraint := constraint.(type) {
	case *ast.TableConstraint_PrimaryKey:
		{
			if !table.Constraints.PK.Node.Eq(constraint) {
				panic("DropConstraint: pk does not match")
			}

			table.Constraints.PK = nil
		}
	case *ast.TableConstraint_ForeignKey:
		{
			table.Constraints.FKs = slices.DeleteFunc(table.Constraints.FKs, func(other *ForeignKey) bool {
				return other.Node.Eq(constraint)
			})
		}
	case *ast.TableConstraint_Check:
		{
			table.Constraints.Checks = slices.DeleteFunc(table.Constraints.Checks, func(other *Check) bool {
				return other.Node.Eq(constraint)
			})
		}
	case *ast.TableConstraint_Unique:
		{
			table.Constraints.Uniques = slices.DeleteFunc(table.Constraints.Uniques, func(other *Unique) bool {
				return other.Node.Eq(constraint)
			})
		}
	}

	return nil
}

func TableFromAst(statement *ast.CreateTable) (*Table, error) {
	// this is where we validate that the table is "correct" by itself
	errs := []error{}

	table := &Table{
		Node:        statement,
		Name:        statement.TableIdentifier.String(),
		Temporary:   statement.Temporary != nil,
		Columns:     map[string]ColumnLike{},
		Constraints: TableConstraints{},
	}

	// add all the columns
	for i := range statement.TableDefinition.ColumnDefinitions {

		colDef := &statement.TableDefinition.ColumnDefinitions[i]

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
			table.Columns[colDef.ColumnName.String()] = column
			err := GeneratedColumnFromAst(column, colDef)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
				continue
			}
			colLike = column
		} else {
			column := &Column{}
			table.Columns[colDef.ColumnName.String()] = column
			err := ColumnFromAst(column, &statement.TableDefinition.ColumnDefinitions[i])
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
				continue
			}
			colLike = column
		}
		// we want to add all of the table relevant constraints that were picked up
		// in the columns, so that we can make sure that they do not conflict with table constraints
		// defined at the table level. so we add them to the table here, in prep for the validation of
		// table constraints
		constraints := colLike.GetConstraints()

		if constraints.PK != nil {
			// a table can only have one primary key
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

	// add all the constraints
	for _, constraint := range statement.TableDefinition.TableConstraints {
		err := table.AddConstraint(constraint)
		if err != nil {
			errs = append(errs, errext.UnwrapAll(err)...)
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
			col, ok := table.Columns[fromCol.String()]
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
			fk.FromColumns[i] = col
		}
	}

	return table, errors.Join(errs...)
}
