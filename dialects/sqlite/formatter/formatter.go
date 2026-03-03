package formatter

import (
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/formatter"
)

type SqliteFormatter struct {
	formatter.Formatter
	ast.BaseVisitor
}

func NewSqliteFormatter(debug bool) *SqliteFormatter {
	return &SqliteFormatter{
		BaseVisitor: ast.BaseVisitor{
			Debug: debug,
		},
	}
}

func (f *SqliteFormatter) VisitStatements(node []ast.Statement) {
	for _, stmt := range node {
		stmt.Accept(f)
		f.Rune(';')
		f.Break()
	}
}

func (f *SqliteFormatter) VisitAlterTable(node *ast.AlterTable) {
	f.Group(func() {
		f.Text("ALTER")
		f.Space()
		f.Text("TABLE")
		f.Space()
		node.TableIdentifier.Accept(f)
		f.Space()
		node.Alteration.Accept(f)
	})
}

func (f *SqliteFormatter) VisitTableAlterationAddColumn(node *ast.AddColumn) {
	f.Text("ADD")
	f.Space()
	f.Text("COLUMN")
	f.Space()
	node.ColumnDefinition.Accept(f)
}

func (f *SqliteFormatter) VisitTableAlterationDropColumn(node *ast.DropColumn) {
	f.Text("DROP")
	f.Space()
	f.Text("COLUMN")
	f.Space()
	f.Identifier(node.ColumnName.Text)
}

func (f *SqliteFormatter) VisitColumnDefinition(node *ast.ColumnDefinition) {
	node.ColumnName.Accept(f)
	if node.TypeName != nil {
		f.Space()
		node.TypeName.Accept(f)
	}
	if len(node.ColumnConstraints) > 0 {
		f.Space()
	}
	for i := range len(node.ColumnConstraints) {
		node.ColumnConstraints[i].Accept(f)
		if i < len(node.ColumnConstraints)-1 {
			f.Space()
		}
	}
}

func (f *SqliteFormatter) VisitCatalogObjectIdentifier(node *ast.CatalogObjectIdentifier) {
	if node.SchemaName != nil {
		node.SchemaName.Accept(f)
		f.Rune('.')
	}
	node.ObjectName.Accept(f)
}

func (f *SqliteFormatter) VisitIdentifier(node *ast.Identifier) {
	f.Identifier(node.Text)
}

func (f *SqliteFormatter) VisitTypeName(node *ast.TypeName) {
	f.Text(node.Name.Text)
}
