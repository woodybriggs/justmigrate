package generator

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"unsafe"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/formatter"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/datastructures"

	"golang.org/x/sys/cpu"
)

var (
	ErrMissingColumns = errors.New("columns do not exist")
)

type MissingColumnsErr struct {
	Table   *ast.CatalogObjectIdentifier
	Columns []ast.Identifier
}

func (uce *MissingColumnsErr) Error() string {
	return ErrMissingColumns.Error()
}

func (uce *MissingColumnsErr) Unwrap() []error {
	errs := []error{}
	for _, col := range uce.Columns {
		err := report.NewReport("invalid foreign key").
			WithLabels(
				[]report.Label{
					{
						Source: uce.Table.ObjectName.SourceCode,
						Range:  uce.Table.ObjectName.SourceRange,
						Note:   fmt.Sprintf("table missing column \"%s\"", col.Text),
					},
					{
						Source: col.SourceCode,
						Range:  col.SourceRange,
						Note:   "column used here",
					},
				},
			).
			WithMessage(fmt.Sprintf("\"%s\" does not exist on table \"%s\"", col.Text, uce.Table.ObjectName.Text)).
			WithNotes([]string{"add the missing column to the table"})

		errs = append(errs, err)
	}
	return errs
}

type SchemaGraph struct {
	Tables                    map[string]*Table
	unresolvedForeignKeyEdges []UnresolvedForeignKeyEdge
}

func NewSchemaGraph() *SchemaGraph {
	return &SchemaGraph{
		Tables:                    map[string]*Table{},
		unresolvedForeignKeyEdges: []UnresolvedForeignKeyEdge{},
	}
}

func (sg *SchemaGraph) Sort(statements []ast.Statement) ([]ast.Statement, error) {
	order, err := sg.sort()
	if err != nil {
		return nil, err
	}

	outputStatements := []ast.Statement{}
	for _, table := range order {
		outputStatements = append(outputStatements, table.CreateTable)
	}
	slices.Reverse(outputStatements)

	core := formatter.NewCoreFormatter(os.Stdout, 80, "``")
	fmtter := NewSqliteFormatter(true, core)
	fmtter.VisitStatements(outputStatements)

	return nil, nil
}

func (sg *SchemaGraph) sort() ([]*Table, error) {
	inDegree := map[string]int{}

	for _, table := range sg.Tables {
		inDegree[table.Name] = 0
	}

	for _, table := range sg.Tables {
		for _, edge := range table.ForeignKeys {
			inDegree[edge.ToTable.Name] += 1
		}
	}

	// not nessecary, but is nice for the data of the Queue to fit into a single cache line
	cacheLineSize := int(unsafe.Sizeof(cpu.CacheLinePad{}))
	ptrSize := int(unsafe.Sizeof(uintptr(0)))
	q := datastructures.NewQueue[*Table](cacheLineSize / ptrSize)
	for name, degree := range inDegree {
		if degree == 0 {
			q.Enqueue(sg.Tables[name])
		}
	}

	order := []*Table{}
	for node, ok := q.Dequeue(); ok; node, ok = q.Dequeue() {
		order = append(order, node)
		for _, edge := range node.ForeignKeys {
			inDegree[edge.ToTable.Name] -= 1
			if inDegree[edge.ToTable.Name] == 0 {
				q.Enqueue(edge.ToTable)
			}
		}
	}

	// if there is a depenency cycle
	if len(order) != len(sg.Tables) {
		// since sqlite allows creating tables with foreign keys to
		// non-existent tables we can just append the remaining tables.
		// in any order
		remaining := []*Table{}
		for _, table := range sg.Tables {
			if inDegree[table.Name] > 0 {
				remaining = append(remaining, table)
			}
		}

		order = append(order, remaining...)
	}

	return order, nil
}

type missingTable struct {
	MissingTableIdentifier ast.CatalogObjectIdentifier
	ReferencedBy           *ast.CreateTable
}

type ErrSchemaResolutionFailed struct {
	MissingTables  []missingTable
	MissingColumns []ast.Identifier
}

func (err *ErrSchemaResolutionFailed) Error() string {
	return "schema resolution failed to resolve some foreign keys"
}

func (srf *ErrSchemaResolutionFailed) Unwrap() []error {

	errs := []error{}

	for _, table := range srf.MissingTables {

		referencedSource := table.ReferencedBy.CreateKeyword.SourceCode
		referencedRange := table.ReferencedBy.TableIdentifier.ObjectName.SourceRange

		err := report.NewReport("foreign key").
			WithLocation(table.MissingTableIdentifier.ObjectName.FileLoc).
			WithLabels([]report.Label{
				{
					Source: table.MissingTableIdentifier.ObjectName.SourceCode,
					Range:  table.MissingTableIdentifier.ObjectName.SourceRange,
					Note:   "this foreign table is missing",
				},
				{
					Source: referencedSource,
					Range:  referencedRange,
					Note:   "forign key",
				},
			})

		errs = append(errs, err)
	}

	for _, col := range srf.MissingColumns {
		err := report.NewReport("invalid foreign key").
			WithLabels([]report.Label{{
				Source: col.SourceCode,
				Range:  col.SourceRange,
				Note:   "here",
			}}).
			WithNotes([]string{"the column referenced does not exist on the foreign table"})

		errs = append(errs, err)
	}

	return errs
}

func (sg *SchemaGraph) Resolve() error {

	missingTables := []missingTable{}
	totalMissingColumns := []ast.Identifier{}

	missingColumns := []ast.Identifier{}
	for _, unresolved := range sg.unresolvedForeignKeyEdges {
		missingColumns = missingColumns[:0]
		toTable, hasTable := sg.TableByIdent(&unresolved.ToTable)
		if !hasTable {
			missingTables = append(missingTables, missingTable{
				MissingTableIdentifier: unresolved.ToTable,
				ReferencedBy:           unresolved.FromTable.CreateTable,
			})
			continue
		}

		toColumns, founds := sg.ColumnsByIdents(toTable, unresolved.ToColumns)
		for i, found := range founds {
			if !found {
				missingColumns = append(missingColumns, unresolved.ToColumns[i])
			}
		}

		if len(missingColumns) > 0 {
			totalMissingColumns = append(totalMissingColumns, missingColumns...)
			continue
		}

		final := &ForeignKeyEdge{
			FromTable:   unresolved.FromTable,
			FromColumns: unresolved.FromColumns,
			ToTable:     toTable,
			ToColumns:   toColumns,
		}
		unresolved.FromTable.ForeignKeys = append(unresolved.FromTable.ForeignKeys, final)
	}

	if len(missingTables) > 0 || len(totalMissingColumns) > 0 {
		return &ErrSchemaResolutionFailed{
			MissingTables:  missingTables,
			MissingColumns: totalMissingColumns,
		}
	}

	return nil
}

func validateColumnsExist(table *ast.CreateTable, idents ast.IdentifierList) (err error) {

	columns := ast.IdentifierList{}
	for _, col := range table.TableDefinition.ColumnDefinitions {
		columns = append(columns, col.ColumnName)
	}

	missingColumns := []ast.Identifier{}
	for _, ident := range idents {
		contained := slices.ContainsFunc(columns, func(item ast.Identifier) bool {
			return ident.AsExpr().Eq(item.AsExpr())
		})
		if !contained {
			missingColumns = append(missingColumns, ident)
		}
	}

	if len(missingColumns) > 0 {
		err = &MissingColumnsErr{
			Table:   table.TableIdentifier,
			Columns: missingColumns,
		}
	}

	return
}

func validateTableConstraints(t *ast.CreateTable) (err error) {
	for _, constraint := range t.TableDefinition.TableConstraints {

		switch c := constraint.(type) {
		case *ast.TableConstraint_ForeignKey:
			// check that the tables.columns in the fk actually exist
			err = validateColumnsExist(t, c.Columns)
			if err != nil {
				return
			}
		default:
			continue
		}
	}

	return
}

func (sg *SchemaGraph) AddTable(t *ast.CreateTable) error {

	err := validateTableConstraints(t)
	if err != nil {
		return err
	}

	table := &Table{
		CreateTable: t,
		Columns:     map[string]*Column{},
	}
	defer func() {
		sg.Tables[t.TableIdentifier.ObjectName.Text] = table
	}()

	table.Name = t.TableIdentifier.ObjectName.Text
	for _, column := range t.TableDefinition.ColumnDefinitions {
		sg.AddColumn(table, &column)
	}

	for _, constraint := range t.TableDefinition.TableConstraints {
		if fk, ok := constraint.(*ast.TableConstraint_ForeignKey); ok {

			// find the foreign table or yield it incase it comes later
			toTable, hasTable := sg.TableByIdent(&fk.FkClause.ForeignTable)
			if !hasTable {
				unresolved := UnresolvedForeignKeyEdge{
					FromTable:   table,
					FromColumns: table.GetColumns(fk.Columns),
					ToTable:     fk.FkClause.ForeignTable,
					ToColumns:   fk.FkClause.ForeignColumns,
				}
				sg.unresolvedForeignKeyEdges = append(sg.unresolvedForeignKeyEdges, unresolved)
				continue
			}

			// validate that the foreign table has the foreign columns
			err = validateColumnsExist(toTable.CreateTable, fk.FkClause.ForeignColumns)
			if err != nil {
				// @todo(woody) this early exists the loop we want to accumulate these errors up to not early exit
				return err
			}

			// add the foreign key edge
			table.AddForeignKeyEdge(
				table.GetColumns(fk.Columns),
				toTable,
				toTable.GetColumns(fk.FkClause.ForeignColumns),
			)
		}
	}
	return nil
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
		Name: col.ColumnName,
		Type: col.TypeName,
	}
	table.Columns[column.Name.Text] = column

	for _, constraint := range col.ColumnConstraints {
		if fk, ok := constraint.(*ast.ColumnConstraint_ForeignKey); ok {

			// find the foreign table, yield if it hasn't been added yet
			toTable, hasTable := sg.TableByIdent(&fk.FkClause.ForeignTable)
			if !hasTable {
				sg.unresolvedForeignKeyEdges = append(sg.unresolvedForeignKeyEdges, UnresolvedForeignKeyEdge{
					FromTable:   table,
					FromColumns: []*Column{column},
					ToTable:     fk.FkClause.ForeignTable,
					ToColumns:   fk.FkClause.ForeignColumns,
				})

				return nil
			}

			err := validateColumnsExist(toTable.CreateTable, fk.FkClause.ForeignColumns)
			if err != nil {
				// @todo(woody) this early exists the loop we want to accumulate these errors up to not early exit
				return err
			}

			table.AddForeignKeyEdge(
				[]*Column{column},
				toTable,
				toTable.GetColumns(fk.FkClause.ForeignColumns),
			)
		}
	}
	return nil
}

func (sg *SchemaGraph) AddIndex(t *ast.CreateIndex) {

}

func (table *Table) AddForeignKeyEdge(cols []*Column, foreignTable *Table, foreignCols []*Column) {
	table.ForeignKeys = append(table.ForeignKeys, &ForeignKeyEdge{
		FromTable:   table,
		FromColumns: cols,
		ToTable:     foreignTable,
		ToColumns:   foreignCols,
	})
}

type Table struct {
	CreateTable *ast.CreateTable

	Name        string
	Columns     map[string]*Column
	Indexes     []*Index
	ForeignKeys []*ForeignKeyEdge
}

func (t *Table) GetColumns(idents []ast.Identifier) []*Column {
	result := []*Column{}

	for _, ident := range idents {
		result = append(result, t.Columns[ident.Text])
	}

	return result
}

type Column struct {
	Name ast.Identifier
	Type *ast.TypeName

	ParentTable *Table
}

type Index struct {
	Name   string
	Table  *Table
	Column []*Column
}

type ForeignKeyEdge struct {
	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []*Column

	// ToTable is the table being referenced (the "parent" table).
	ToTable   *Table
	ToColumns []*Column
}

type UnresolvedForeignKeyEdge struct {
	// FromTable is the table defining the foreign key constraint (the "child" table).
	FromTable   *Table
	FromColumns []*Column

	// ToTable is the table being referenced (the "parent" table).
	ToTable   ast.CatalogObjectIdentifier
	ToColumns []ast.Identifier
}
