package sqlite

import (
	"io"
	"os"
	"slices"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/tik"
	sqliteformatter "woodybriggs/justmigrate/dialects/sqlite/formatter"
	"woodybriggs/justmigrate/diff"
	"woodybriggs/justmigrate/formatter"
)

type SqliteGenerator struct {
	edits []diff.Edit
}

func NewSqliteGenerator(edits []diff.Edit) *SqliteGenerator {
	return &SqliteGenerator{
		edits: edits,
	}
}

func (gen *SqliteGenerator) Generate(writer io.Writer) {

	statements := []ast.Statement{}

	for _, edit := range gen.edits {
		switch typ := edit.(type) {
		case *diff.EditAddTable:
			{
				statements = append(statements, typ.CreateTable)
			}
		case *diff.EditRemoveTable:
			{
				statements = append(statements, dropTable(typ.TableIdentifier))
			}
		case *diff.EditModifyTable:
			{
				statements = slices.Concat(statements, alterTable(typ.Target, typ.Edits))
			}

		default:
			{
				panic("not implemented")
			}
		}
	}

	core := formatter.NewCoreFormatter(os.Stderr, 80, "\"\"")
	form := &sqliteformatter.SqliteFormatter{Formatter: core, BaseVisitor: ast.BaseVisitor{Debug: true}}
	form.VisitStatements(statements)
}

func alterTable(table *ast.CreateTable, edits []diff.Edit) []ast.Statement {

	statements := []ast.Statement{}

	for _, edit := range edits {
		switch typ := edit.(type) {
		case *diff.EditAddColumn:
			{
				statements = append(statements, alterTableAddColumn(table, typ.ColumnDefinition))
			}
		case *diff.EditRemoveColumn:
			{
				statements = append(statements, alterTableDropColumn(table, typ.ColumnDefinition))
			}
		}
	}

	return statements
}

func alterTableAddColumn(table *ast.CreateTable, column ast.ColumnDefinition) *ast.AlterTable {
	return &ast.AlterTable{
		AlterKeyword: ast.Keyword(
			tik.Token{
				Text: "ALTER",
				Kind: tik.TokenKind_Keyword_ALTER,
			},
		),
		TableKeyword: ast.Keyword(
			tik.Token{
				Text: "TABLE",
				Kind: tik.TokenKind_Keyword_TABLE,
			},
		),
		TableIdentifier: table.TableIdentifier,
		Alteration: &ast.AddColumn{
			AddKeyword: ast.Keyword(
				tik.Token{
					Text: "ADD",
					Kind: tik.TokenKind_Keyword_ADD,
				},
			),
			ColumnDefinition: column,
		},
	}
}

func alterTableDropColumn(table *ast.CreateTable, column ast.ColumnDefinition) *ast.AlterTable {
	return &ast.AlterTable{
		AlterKeyword: ast.Keyword(
			tik.Token{
				Text:           "ALTER",
				TrailingTrivia: " ",
				Kind:           tik.TokenKind_Keyword_ALTER,
			},
		),
		TableKeyword: ast.Keyword(
			tik.Token{
				Text:           "TABLE",
				TrailingTrivia: " ",
				Kind:           tik.TokenKind_Keyword_TABLE,
			},
		),
		TableIdentifier: table.TableIdentifier,
		Alteration: &ast.DropColumn{
			DropKeyword: ast.Keyword(tik.Token{
				Text: "DROP",
				Kind: tik.TokenKind_Keyword_DROP,
			}),
			ColumnName: column.ColumnName,
		},
	}
}

func dropTable(tableIdentifier *ast.CatalogObjectIdentifier) *ast.DropTable {
	return &ast.DropTable{
		IfExists: &ast.IfExists{
			If: ast.Keyword(tik.Token{
				Text: "IF",
			}),
			Exists: ast.Keyword(tik.Token{
				Text: "EXISTS",
			}),
		},
		TableIdentifier: *tableIdentifier,
	}
}
