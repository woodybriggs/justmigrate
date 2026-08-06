package generator

import (
	"errors"
	"fmt"
	"justmigrate/internal/backend/op"
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/errext"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/report"
	"justmigrate/internal/frontend/token"
	"justmigrate/internal/sliceext"
)

/*

translator is responsible for converting ops and schema objects into ast.Statement
for the generator to be able to output the final sql text

*/

func Translate(schem *schema.Schema, ops []op.Op) ([]ast.Statement, error) {

	errs := []error{}
	result := []ast.Statement{}

	state := schem.Clone()

	for _, o := range ops {
		switch o := o.(type) {
		case *op.AddTable:
			state.Tables[o.Table.Name] = o.Table
			stmt := schema.CreateTableAstFromTable(nil, o.Table)
			result = append(result, stmt)
		case *op.DropTable:
			state.Tables[o.Target.Table.Name] = nil
			stmt := schema.DropTableAstFromTable(nil, o.Target.Table)
			result = append(result, stmt)
		case *op.RenameTable:
			state.Tables[o.NewName.ObjectName.String()] = o.Target.Table.Clone()
			state.Tables[o.NewName.ObjectName.String()].Name = o.NewName.ObjectName.String()
			delete(state.Tables, o.Target.Table.Name)
			stmt := alterTableRename(o)
			result = append(result, stmt)
		case *op.MigrateData:
			stmt, err := insertIntoMigrateData(schem, o.SrcTarget.Table, o.DstTarget.Table, o.Ops)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			} else {
				result = append(result, stmt)
			}
		case *op.AddColumn:
			state.Tables[o.Target.Table.Name].Columns[o.Column.GetName().String()] = o.Column
			state.Columns[o.Target.Table.Name][o.Column.GetName().String()] = o.Column
			stmt := alterTableAddColummn(o)
			result = append(result, stmt)
		default:
			fmt.Printf("unhandled op in translator %T\n", o)
		}
	}

	return result, nil
}

func alterTableRename(o *op.RenameTable) *ast.AlterTable {
	// @TODO(woody): again we should probably store the catalog object identifier in schema.Table
	var schemaName *ast.Identifier = nil
	// if o.Target.Schema.Name != "" {
	// 	schemaName = ast.MakeIdentifier(token.Identifier(o.Target.Schema.Name))
	// }
	oldTableIdent := ast.MakeCatalogObjectIdentifier(
		schemaName,
		ast.Identifier(token.Identifier(o.Target.Table.Name)),
	)

	return ast.MakeAlterTable(
		ast.Keyword(token.Keyword("ALTER")),
		ast.Keyword(token.Keyword("TABLE")),
		oldTableIdent,
		ast.MakeRenameTableAlteration(
			ast.Keyword(token.Keyword("RENAME")),
			ast.Keyword(token.Keyword("TO")),
			o.NewName,
		),
	)
}

func insertIntoMigrateData(schem *schema.Schema, oldTable, newTable *schema.Table, ops []op.Op) (*ast.InsertInto, error) {

	errs := []error{}

	insertCols := []ast.Identifier{}
	selectExprs := []ast.Expr{}

	for _, newCol := range newTable.Columns {
		renameTo, found := sliceext.FindFunc(ops, func(o op.Op) bool {
			re, ok := o.(*op.RenameColumn)
			if !ok {
				return false
			}
			return re.NewName.Eq(newCol.GetName())
		})

		if found {
			insertCols = append(insertCols, *renameTo.(*op.RenameColumn).NewName)
			selectExprs = append(selectExprs, renameTo.(*op.RenameColumn).Target.Column.GetName())
			continue
		}

		oldCol, found := oldTable.Columns[newCol.GetName().String()]
		if found {
			insertCols = append(insertCols, *oldCol.GetName())
			selectExprs = append(selectExprs, oldCol.GetName())
			continue
		}

		if newCol.GetConstraints().Default != nil {
			insertCols = append(insertCols, *newCol.GetName())
			selectExprs = append(selectExprs, newCol.GetConstraints().Default.Expr)
			continue
		}

		if newCol.GetConstraints().NotNull == nil {
			insertCols = append(insertCols, *newCol.GetName())
			selectExprs = append(selectExprs, ast.MakeLiteralNull(token.Token{Text: "NULL", Kind: token.TokenKind_Keyword_NULL}))
			continue
		}

		// column has not null constraint with no default value
		err := report.
			NewReport("planning error").
			WithLocation(newCol.GetName().FileLoc).
			WithMessage("newly added column has constraint 'not null' but does not define a 'default' value").
			WithLabels(
				report.LabelFromIdentifier(*newCol.GetName(), "remove 'not null' constraint or add a 'default' value"),
			)
		errs = append(errs, err)
	}

	insertInto := ast.MakeInsertInto(
		ast.Keyword(token.Keyword("INSERT")),
		nil,
		ast.Keyword(token.Keyword("INTO")),
		newTable.CatalogObjectIdentifier(),
		insertCols,
		&ast.InsertIntoValuesSelect{
			Select: ast.MakeSelectFromTable(
				ast.Keyword(token.Keyword("SELECT")),
				selectExprs,
				oldTable.CatalogObjectIdentifier(),
				nil,
			),
			UpsertClauses: nil,
		},
	)

	return insertInto, errors.Join(errs...)
}

func alterTableAddColummn(o *op.AddColumn) ast.Statement {
	return ast.MakeAlterTable(
		ast.Keyword(token.Keyword("ALTER")),
		ast.Keyword(token.Keyword("TABLE")),
		o.Target.Table.CatalogObjectIdentifier(),
		&ast.AddColumn{
			AddKeyword:       ast.Keyword(token.Keyword("ADD")),
			ColumnKeyword:    ast.MakeKeyword(token.Keyword("COLUMN")),
			ColumnDefinition: *schema.ColumnDefinitionAstFromColumn(o.Column),
		},
	)
}
