package generator

import (
	"errors"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/report"
)

var (
	ErrUndefinedColumns = errors.New("columns are not defined")
)

type UndefinedColumnsErr struct {
	UndefinedColumns []ast.Identifier
}

func (uce *UndefinedColumnsErr) Error() string {
	return ErrUndefinedColumns.Error()
}

func (uce *UndefinedColumnsErr) Unwrap() []error {
	errs := []error{}
	for _, col := range uce.UndefinedColumns {
		err := report.NewReport("type check").
			WithLabels([]report.Label{{
				Source: col.SourceCode,
				Range:  col.SourceRange,
				Note:   "here",
			}}).
			WithMessage("the column referenced here does not exist on the forign table")

		errs = append(errs, err)
	}
	return errs
}

type UnresolvedForeignKeyEdge struct {
	Name string
	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []*Column

	// ToTable is the table being referenced (the "parent" table).
	ToTable   ast.CatalogObjectIdentifier
	ToColumns []ast.Identifier
}

type SchemaGraph struct {
	Tables                    map[string]*Table
	unresolvedForeignKeyEdges []UnresolvedForeignKeyEdge
}

func (sg *SchemaGraph) AddTable(t *ast.CreateTable) error {
	table := &Table{}

	table.Name = t.TableIdentifier.ObjectName.Text
	for _, column := range t.TableDefinition.ColumnDefinitions {
		sg.AddColumn(table, &column)
	}

	for _, constraint := range t.TableDefinition.TableConstraints {
		if fk, ok := constraint.(*ast.TableConstraint_ForeignKey); ok {

		}
	}

	sg.Tables[t.TableIdentifier.ObjectName.Text] = table
	return nil
}

func constraintNameOrEmptyString(cName *ast.ConstraintName) string {
	if cName != nil {
		return cName.Name.Text
	}
	return ""
}

func (sg *SchemaGraph) TableByIdent(ident *ast.CatalogObjectIdentifier) (*Table, bool) {
	if t, has := sg.Tables[ident.ObjectName.Text]; has {
		return t, true
	}
	return nil, false
}

func (sg *SchemaGraph) ColumnsByIdents(table *Table, idents []ast.Identifier) ([]*Column, []bool) {
	result := []*Column{}
	found := []bool{}

	for _, ident := range idents {
		if col, has := table.Columns[ident.Text]; has {
			result = append(result, col)
			found = append(found, true)
		} else {
			found = append(found, false)
		}
	}

	return result, found
}

func (sg *SchemaGraph) AddColumn(table *Table, col *ast.ColumnDefinition) error {

	column := &Column{
		Name: col.ColumnName.Text,
		Type: col.TypeName,
	}
	table.Columns[column.Name] = column

	for _, constraint := range col.ColumnConstraints {
		if fk, ok := constraint.(*ast.ColumnConstraint_ForeignKey); ok {

			toTable, hasTable := sg.TableByIdent(&fk.FkClause.ForeignTable)
			if !hasTable {
				sg.unresolvedForeignKeyEdges = append(sg.unresolvedForeignKeyEdges, UnresolvedForeignKeyEdge{
					Name:        constraintNameOrEmptyString(fk.Name),
					FromTable:   table,
					FromColumns: []*Column{column},
					ToTable:     fk.FkClause.ForeignTable,
					ToColumns:   fk.FkClause.ForeignColumns,
				})
			}

			toColumns, founds := sg.ColumnsByIdents(toTable, fk.FkClause.ForeignColumns)

			uce := UndefinedColumnsErr{}
			for i, found := range founds {
				if !found {
					uce.UndefinedColumns = append(uce.UndefinedColumns, fk.FkClause.ForeignColumns[i])
				}
			}
			if len(uce.UndefinedColumns) > 0 {
				return &uce
			}

			foreignKey := &ForeignKeyEdge{
				Name:        constraintNameOrEmptyString(fk.Name),
				FromTable:   table,
				FromColumns: []*Column{column},
				ToTable:     toTable,
				ToColumns:   toColumns,
			}

			table.ForiegnKeys = append(table.ForiegnKeys, foreignKey)
		}
	}
	return nil
}

func (sg *SchemaGraph) AddIndex(t *ast.CreateIndex) {

}

func (table *Table) AddForeignKeyEdge(cols []*Column, foreignTable *Table, foreignCols []*Column) {

}

type Table struct {
	Name string

	Columns     map[string]*Column
	Indexes     []*Index
	ForiegnKeys []*ForeignKeyEdge
}

type Column struct {
	Name string
	Type *ast.TypeName

	ParentTable *Table
}

type Index struct {
	Name   string
	Table  *Table
	Column []*Column
}

type ForeignKeyEdge struct {
	Name string

	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []*Column

	// ToTable is the table being referenced (the "parent" table).
	ToTable   *Table
	ToColumns []*Column
}
