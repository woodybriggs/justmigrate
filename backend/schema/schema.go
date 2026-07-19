package schema

import (
	"errors"
	"fmt"
	"maps"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/report"
)

type schemaBuilder struct {
	SchemaName  string
	Tables      map[string]*Table
	Columns     map[string]map[string]ColumnLike
	ForeignKeys []*ForeignKey

	unresolvedForeignKeyEdges []*ForeignKey
	errors                    []error
}

func (builder *schemaBuilder) Schema() (*Schema, error) {
	// resolve fks here
	// we only need to resolve ToTable ToColumns
	// FromTable and FromColumns are validated in TableFromAst
	for _, uFK := range builder.unresolvedForeignKeyEdges {
		foreignTable, ok := builder.Tables[uFK.Unresolved.ToTable.String()]
		if !ok {
			// error referenced table does not exist in schema
			err := report.
				NewReport("invalid foreign key definition").
				WithLocation(uFK.Unresolved.ToTable.ObjectName.FileLoc).
				WithLabels(
					report.LabelFromIdentifier(uFK.Unresolved.ToTable.ObjectName, "referenced table does not exist in schema"),
				)
			builder.errors = append(builder.errors, err)
			continue
		}

		uFK.ToTable = foreignTable

		uFK.ToColumns = make([]ColumnLike, len(uFK.Unresolved.ToColumns))
		for j, colID := range uFK.Unresolved.ToColumns {
			foreignCol, ok := foreignTable.Columns[colID.String()]
			if !ok {
				// error referended column does not exist in foreign table
				err := report.
					NewReport("invalid foreign key definition").
					WithLocation(colID.FileLoc).
					WithLabels(
						report.LabelFromIdentifier(colID, "this column does not exist in the referenced table"),
						report.LabelFromIdentifier(foreignTable.Node.TableIdentifier.ObjectName, "referenced table"),
					)
				builder.errors = append(builder.errors, err)
				continue
			}

			uFK.ToColumns[j] = foreignCol
		}

		builder.ForeignKeys = append(builder.ForeignKeys, uFK)
	}

	return &Schema{
		Name:        builder.SchemaName,
		Tables:      builder.Tables,
		Columns:     builder.Columns,
		ForeignKeys: builder.ForeignKeys,
	}, errors.Join(builder.errors...)
}

func (builder *schemaBuilder) AddTable(table *Table) {
	// this is where we validate that the table is correct within the schema
	// and place any unresovled items in the builder for later resolution

	builder.Tables[table.Name] = table
	builder.Columns[table.Name] = table.Columns

	// here we try to resolve foreign keys, and add them to unresolved list if a table is not in scope yet
	for i, fk := range table.Constraints.FKs {
		// lookup foreign table reference
		foreignTable, ok := builder.Tables[fk.Unresolved.ToTable.String()]
		if !ok {
			// add the fk to unresolved list, to resolve later
			builder.unresolvedForeignKeyEdges = append(builder.unresolvedForeignKeyEdges, fk)
			continue
		}

		// we have a foreign, table so we can update the table's fk
		table.Constraints.FKs[i].ToTable = foreignTable

		table.Constraints.FKs[i].ToColumns = make([]ColumnLike, len(table.Constraints.FKs[i].Unresolved.ToColumns))
		for j, toCol := range table.Constraints.FKs[i].Unresolved.ToColumns {
			// look up the col in the foreign table
			foreignCol, ok := foreignTable.Columns[toCol.Text]
			if !ok {
				// this is now a semantic error
				// the user has referenced a column in a table that does not exist
				err := report.
					NewReport("invalid foreign key definition").
					WithLocation(toCol.FileLoc).
					WithLabels(
						report.LabelFromIdentifier(foreignTable.Node.TableIdentifier.ObjectName, fmt.Sprintf("this referenced table does not define column '%v'", toCol.Text)),
						report.LabelFromIdentifier(toCol, "this column does not exist in the referenced table"),
					)
				builder.errors = append(builder.errors, err)
				continue
			}

			// we have a matching foreignCol, lets map it
			table.Constraints.FKs[i].ToColumns[j] = foreignCol
		}

		// we now have a fully resolved FK in the table, we can add it to the draft schema in builder
		builder.ForeignKeys = append(builder.ForeignKeys, table.Constraints.FKs[i])
	}
}

type Schema struct {
	Name        string
	Tables      map[string]*Table
	Indexes     map[string]*Index
	Columns     map[string]map[string]ColumnLike
	ForeignKeys []*ForeignKey
}

func (schema *Schema) Eq(otherAny any) bool {
	other, ok := otherAny.(*Schema)
	if !ok {
		return false
	}

	tablesEq := maps.EqualFunc(schema.Tables, other.Tables, func(a, b *Table) bool {
		return a.Eq(b)
	})

	indexesEq := maps.EqualFunc(schema.Indexes, other.Indexes, func(a, b *Index) bool {
		return a.Eq(b)
	})

	return tablesEq && indexesEq
}

func (schema *Schema) Clone() *Schema {

	builder := &schemaBuilder{
		SchemaName:  schema.Name,
		Tables:      map[string]*Table{},
		Columns:     map[string]map[string]ColumnLike{},
		ForeignKeys: make([]*ForeignKey, 0),
	}

	for _, table := range schema.Tables {
		builder.AddTable(table.Clone())
	}

	cloned, err := builder.Schema()
	if err != nil {
		panic(err)
	}
	return cloned
}

func SchemaFromAst(name string, statements []ast.Statement) (*Schema, error) {

	builder := &schemaBuilder{
		SchemaName:  name,
		Tables:      map[string]*Table{},
		Columns:     map[string]map[string]ColumnLike{},
		ForeignKeys: make([]*ForeignKey, 0),
	}

	for _, statement := range statements {
		switch stmt := statement.(type) {
		case *ast.CreateTable:
			{
				table, err := TableFromAst(stmt)
				if err != nil {
					builder.errors = append(builder.errors, err)
					continue
				}

				builder.AddTable(table)
			}
		}
	}

	return builder.Schema()
}
