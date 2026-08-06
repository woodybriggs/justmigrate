package schema

import (
	"errors"
	"fmt"
	"slices"
	"justmigrate/internal/errext"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/report"
	"justmigrate/internal/frontend/token"
)

type TableConstraints struct {
	PK      *PrimaryKey
	FKs     []*ForeignKey
	Checks  []*Check
	Uniques []*Unique
}

func (tc TableConstraints) Clone() TableConstraints {
	clone := TableConstraints{}

	if tc.PK != nil {
		pk := *tc.PK
		pk.IndexedColumns = slices.Clone(tc.PK.IndexedColumns)
		clone.PK = &pk
	}

	if tc.FKs != nil {
		clone.FKs = make([]*ForeignKey, len(tc.FKs))
		for i, fk := range tc.FKs {
			f := *fk
			clone.FKs[i] = &f
		}
	}

	if tc.Checks != nil {
		clone.Checks = make([]*Check, len(tc.Checks))
		for i, check := range tc.Checks {
			c := *check
			clone.Checks[i] = &c
		}
	}

	if tc.Uniques != nil {
		clone.Uniques = make([]*Unique, len(tc.Uniques))
		for i, unique := range tc.Uniques {
			u := *unique
			clone.Uniques[i] = &u
		}
	}

	return clone
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

func (table *Table) ResolveInternalReferences() error {
	errs := []error{}

	for _, fk := range table.Constraints.FKs {
		fk.FromTable = table

		fk.FromColumns = make([]ColumnLike, len(fk.Unresolved.FromColumns))
		for i, fromCol := range fk.Unresolved.FromColumns {
			col, ok := table.Columns[fromCol.String()]
			if !ok {
				err := report.
					NewReport("invalid foreign key definition").
					WithLocation(fromCol.FileLoc).
					WithLabels(
						report.LabelFromIdentifier(table.Node.TableIdentifier.ObjectName, fmt.Sprintf("this table does not define column '%v'", fromCol.Text)),
						report.LabelFromIdentifier(fromCol, "this column does not exist"),
					)

				errs = append(errs, err)
				continue
			}

			fk.FromColumns[i] = col
		}
	}

	if table.Constraints.PK != nil {
		pk := table.Constraints.PK
		pk.ResolvedColumns = make([]ColumnLike, len(pk.IndexedColumns))
		for i, indexedCol := range pk.IndexedColumns {
			ident, ok := indexedCol.Subject.(*ast.Identifier)
			if !ok {
				errs = append(errs, report.
					NewReport("semantic error").
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
					NewReport("semantic error").
					WithLocation(report.LocationFromExpr(indexedCol.Subject)).
					WithMessage("column does not exist in table").
					WithLabels(
						report.LabelFromIdentifier(table.Node.TableIdentifier.ObjectName, "this table"),
						report.LabelFromExpr(indexedCol.Subject, "does not declare this column"),
					),
				)
				continue
			}

			pk.ResolvedColumns[i] = col
		}
	}

	return errors.Join(errs...)
}

func (t *Table) Clone() *Table {
	clone := &Table{
		Node:        t.Node,
		Name:        t.Name,
		Temporary:   t.Temporary,
		Columns:     make(map[string]ColumnLike, len(t.Columns)),
		Constraints: t.Constraints.Clone(),
		Indexes:     slices.Clone(t.Indexes),
	}

	for name, col := range t.Columns {
		switch c := col.(type) {
		case *Column:
			clone.Columns[name] = c.Clone()
		case *GeneratedColumn:
			clone.Columns[name] = c.Clone()
		}
	}

	if err := clone.ResolveInternalReferences(); err != nil {
		panic(fmt.Sprintf("resolveInternalReferences failed during clone: %v", err))
	}

	return clone
}

func (t *Table) CatalogObjectIdentifier() *ast.CatalogObjectIdentifier {
	var schemaName *ast.Identifier = nil

	return ast.MakeCatalogObjectIdentifier(
		schemaName,
		*ast.MakeIdentifier(token.Identifier(t.Name)),
	)
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

	// now that the table has been fully defined, we need to resolve the internal wiring of fk and pk
	if err := table.ResolveInternalReferences(); err != nil {
		errs = append(errs, errext.UnwrapAll(err)...)
	}

	return table, errors.Join(errs...)
}

func CreateTableAstFromTable(schemaName *ast.Identifier, table *Table) *ast.CreateTable {

	var tempKeyword *ast.Keyword = nil
	if table.Temporary {
		tempKeyword = ast.MakeKeyword(token.Token{
			Text: "TEMPORARY",
			Kind: token.TokenKind_Keyword_TEMPORARY,
		})
	}

	// @TODO(woody): we don't store if not exists anywhere in schema at the moment
	// and we can't rely on the CreateTable node referenced in the schema struct
	// because this maybe a clone that has deviated from the orignial syntax tree
	var ifNotExists *ast.IfNotExists = nil
	if false {
		ifNotExists = ast.MakeIfNotExists(
			ast.Keyword(token.Token{Text: "IF", Kind: token.TokenKind_Keyword_IF}),
			ast.Keyword(token.Token{Text: "NOT", Kind: token.TokenKind_Keyword_NOT}),
			ast.Keyword(token.Token{Text: "EXISTS", Kind: token.TokenKind_Keyword_EXISTS}),
		)
	}

	var tableOptions *ast.TableOptions = nil
	if false {
		tableOptions = ast.MakeTableOptions(
			ast.MakeKeyword(token.Token{Text: "STRICT", Kind: token.TokenKind_Keyword_STRICT}),
			ast.MakeWithoutRowId(
				ast.Keyword(token.Token{Text: "WITHOUT", Kind: token.TokenKind_Keyword_WITHOUT}),
				ast.Keyword(token.Token{Text: "ROWID", Kind: token.TokenKind_Keyword_ROWID}),
			),
		)
	}

	var tableDefinition *ast.TableDefinition = TableDefinitionAstFromTable(table)

	return ast.MakeCreateTable(
		ast.Keyword(token.Token{Text: "CREATE", Kind: token.TokenKind_Keyword_CREATE}),
		tempKeyword,
		ast.Keyword(token.Token{Text: "TABLE", Kind: token.TokenKind_Keyword_TABLE}),
		ifNotExists,
		ast.MakeCatalogObjectIdentifier(
			schemaName,
			ast.Identifier(token.Token{Text: table.Name, Kind: token.TokenKind_Identifier}),
		),
		tableDefinition,
		tableOptions,
	)
}

func TableDefinitionAstFromTable(table *Table) *ast.TableDefinition {

	columnDefinitions := []ast.ColumnDefinition{}
	for _, column := range table.Columns {
		colDef := ColumnDefinitionAstFromColumn(column)
		columnDefinitions = append(columnDefinitions, *colDef)
	}

	tableConstraints := TableConstraintsAstFromTableConstraints(table.Constraints)

	return &ast.TableDefinition{
		LParen:            token.Token{Text: "(", Kind: token.TokenKind_LParen},
		ColumnDefinitions: columnDefinitions,
		TableConstraints:  tableConstraints,
		RParent:           token.Token{Text: ")", Kind: token.TokenKind_RParen},
	}
}

func TableConstraintsAstFromTableConstraints(constraints TableConstraints) []ast.TableConstraint {
	result := []ast.TableConstraint{}

	if constraints.PK != nil {

		// only if its a table constraint
		if _, ok := constraints.PK.Node.(*ast.TableConstraint_PrimaryKey); ok {

			var constraintName *ast.ConstraintName = nil
			if constraints.PK.Name != nil {
				constraintName = &ast.ConstraintName{
					ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
					Name:              *constraints.PK.Name,
				}
			}

			// @TODO(woody): we don't store autoincrement for sqlite anywhere
			var autoIncrement *ast.Keyword = nil

			result = append(result, ast.MakeTableConstraintPrimaryKey(
				constraintName,
				ast.Keyword(token.Keyword("primary")),
				ast.Keyword(token.Keyword("key")),
				token.Token{Text: "(", Kind: token.TokenKind_LParen},
				constraints.PK.IndexedColumns,
				token.Token{Text: ")", Kind: token.TokenKind_RParen},
				ConflictClauseAstFromConflictAction(constraints.PK.ConflictClause),
				autoIncrement,
			))
		}
	}

	for _, fk := range constraints.FKs {

		// skip if it originated from a column constraint
		if _, ok := fk.Node.(*ast.ColumnConstraint_ForeignKey); ok {
			continue
		}

		var constraintName *ast.ConstraintName = nil
		if fk.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *fk.Name,
			}
		}
		result = append(result, ast.MakeTableConstraintForeignKey(
			constraintName,
			ast.Keyword(token.Keyword("foreign")),
			ast.Keyword(token.Keyword("key")),
			token.Token{Text: "(", Kind: token.TokenKind_LParen},
			fk.Unresolved.FromColumns,
			token.Token{Text: ")", Kind: token.TokenKind_RParen},
			ForeignKeyClauseAstFromForeignKey(fk),
		))
	}

	for _, check := range constraints.Checks {

		// skip if it originated from a column constraint
		if _, ok := check.Node.(*ast.ColumnConstraint_Check); ok {
			continue
		}

		var constraintName *ast.ConstraintName = nil
		if check.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *check.Name,
			}
		}
		result = append(result, ast.MakeTableConstraintCheck(
			constraintName,
			ast.Keyword(token.Keyword("check")),
			token.Token{Text: "(", Kind: token.TokenKind_LParen},
			check.Expr,
			token.Token{Text: ")", Kind: token.TokenKind_RParen},
		))
	}

	for _, unique := range constraints.Uniques {

		// skip if it originated from a column constraint
		if _, ok := unique.Node.(*ast.ColumnConstraint_Unique); ok {
			continue
		}

		var constraintName *ast.ConstraintName = nil
		if unique.Name != nil {
			constraintName = &ast.ConstraintName{
				ConstraintKeyword: ast.Keyword(token.Keyword("CONSTRAINT")),
				Name:              *unique.Name,
			}
		}
		result = append(result, ast.MakeTableConstraintUnique(
			constraintName,
			token.Token{Text: "(", Kind: token.TokenKind_LParen},
			unique.IndexedColumns,
			token.Token{Text: ")", Kind: token.TokenKind_RParen},
			ConflictClauseAstFromConflictAction(unique.ConflictAction),
		))
	}

	return result
}

func DropTableAstFromTable(schema *Schema, table *Table) ast.Statement {

	var schemaName *ast.Identifier = nil
	if schema != nil && schema.Name != "" {
		schemaName = ast.MakeIdentifier(token.Token{Text: schema.Name, Kind: token.TokenKind_Identifier})
	}

	return ast.MakeDropTable(
		nil,
		ast.CatalogObjectIdentifier{
			SchemaName: schemaName,
			ObjectName: ast.Identifier(token.Token{Text: table.Name, Kind: token.TokenKind_Identifier}),
		},
	)
}
