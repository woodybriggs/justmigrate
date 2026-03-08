package generator

import (
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/formatter"
)

type SqliteGenerator struct {
	formatter.Formatter
	ast.BaseVisitor
}

func NewSqliteFormatter(debug bool, formatter formatter.Formatter) *SqliteGenerator {
	return &SqliteGenerator{
		BaseVisitor: ast.BaseVisitor{
			Debug: debug,
		},
		Formatter: formatter,
	}
}

func (f *SqliteGenerator) VisitStatements(node []ast.Statement) {
	for _, stmt := range node {
		stmt.Accept(f)
		f.Rune(';')
		f.Break()
		f.Break()
	}
}

func (f *SqliteGenerator) VisitCreateTable(node *ast.CreateTable) {
	f.Group(func() {
		f.Text("CREATE")
		f.Space()
		f.Text("TABLE")
		f.Space()

		if node.IfNotExist != nil {
			f.Text("IF")
			f.Space()
			f.Text("NOT")
			f.Space()
			f.Text("EXISTS")
			f.Space()
		}

		node.TableIdentifier.Accept(f)
		f.Space()

		f.Rune('(')
		f.Break()
		f.Indent(func() {
			for i, col := range node.TableDefinition.ColumnDefinitions {
				col.Accept(f)
				if i != len(node.TableDefinition.ColumnDefinitions)-1 {
					f.Rune(',')
					f.Break()
				}
			}

			if len(node.TableDefinition.TableConstraints) > 0 {
				f.Rune(',')
				f.Break()
			}

			for i, constraint := range node.TableDefinition.TableConstraints {
				constraint.Accept(f)
				if i < len(node.TableDefinition.TableConstraints)-1 {
					f.Rune(',')
					f.Break()
				}
			}
		})

		f.Break()
		f.Rune(')')

		// visit table options
	})
}

func (f *SqliteGenerator) VisitAlterTable(node *ast.AlterTable) {
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

func (f *SqliteGenerator) VisitTableAlterationAddColumn(node *ast.AddColumn) {
	f.Text("ADD")
	f.Space()
	f.Text("COLUMN")
	f.Space()
	node.ColumnDefinition.Accept(f)
}

func (f *SqliteGenerator) VisitTableAlterationDropColumn(node *ast.DropColumn) {
	f.Text("DROP")
	f.Space()
	f.Text("COLUMN")
	f.Space()
	f.Identifier(node.ColumnName.Text)
}

func (f *SqliteGenerator) VisitColumnDefinition(node *ast.ColumnDefinition) {
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

func (f *SqliteGenerator) VisitCatalogObjectIdentifier(node *ast.CatalogObjectIdentifier) {
	if node.SchemaName != nil {
		node.SchemaName.Accept(f)
		f.Rune('.')
	}
	node.ObjectName.Accept(f)
}

func (f *SqliteGenerator) VisitIdentifier(node *ast.Identifier) {
	f.Identifier(node.Text)
}

func (f *SqliteGenerator) VisitTypeName(node *ast.TypeName) {
	f.Text(node.Name.Text)
	if node.Arg0 != nil {
		f.Rune('(')
		node.Arg0.Accept(f)

		if node.Arg1 != nil {
			f.Rune(',')
			f.Space()
			node.Arg1.Accept(f)
		}

		f.Rune(')')
	}
}

func (f *SqliteGenerator) VisitColumnConstraintNotNull(node *ast.ColumnConstraint_NotNull) {
	f.Text("NOT")
	f.Space()
	f.Text("NULL")
}

func (f *SqliteGenerator) VisitColumnConstraintPrimaryKey(node *ast.ColumnConstraint_PrimaryKey) {
	if node.Name != nil {
		f.Text("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Text("PRIMARY")
	f.Space()
	f.Text("KEY")

	if node.Order != nil {
		f.Space()
		f.Text(node.Order.Text)
		f.Space()
	}

	if node.ConflictClause != nil {
		f.Space()
		f.Text("ON")
		f.Space()
		f.Text("CONFLICT")
		f.Space()
		f.Text(node.ConflictClause.Action.Text)
		f.Space()
	}

	if node.AutoIncrement != nil {
		f.Space()
		f.Text("AUTOINCREMENT")
		f.Space()
	}
}

func (f *SqliteGenerator) VisitColumnConstraintForeignKey(node *ast.ColumnConstraint_ForeignKey) {
	if node.Name != nil {
		f.Text("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.VisitForeignKeyClause(&node.FkClause)
}

func (f *SqliteGenerator) VisitTableConstraintPrimaryKey(node *ast.TableConstraint_PrimaryKey) {
	if node.Name != nil {
		f.Text("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Text("PRIMARY")
	f.Space()
	f.Text("KEY")
	f.Space()

	f.Rune('(')
	if node.AutoIncrement != nil {
		f.VisitIndexedColumn(&node.IndexedColumns[0])
		f.Space()
		f.Text("AUTOINCREMENT")
	} else {
		for i, indexedCol := range node.IndexedColumns {

			f.VisitIndexedColumn(&indexedCol)

			if i != len(node.IndexedColumns)-1 {
				f.Rune(',')
				f.Space()
			}
		}
	}
	f.Rune(')')

	if node.ConflictClause != nil {
		f.Space()
		f.Text("ON")
		f.Space()
		f.Text("CONFLICT")
		f.Space()
		f.Text(node.ConflictClause.Action.Text)
		f.Space()
	}
}

func (f *SqliteGenerator) VisitIndexedColumn(node *ast.IndexedColumn) {
	node.Subject.Accept(f)

	if node.Collation != nil {
		f.Space()
		f.Text("COLLATE")
		f.Space()
		node.Collation.Name.Accept(f)
	}

	if node.Order != nil {
		f.Space()
		f.Text(node.Order.Text)
	}
}

func (f *SqliteGenerator) VisitTableConstraintForeignKey(node *ast.TableConstraint_ForeignKey) {
	if node.Name != nil {
		f.Text("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Text("FOREIGN")
	f.Space()
	f.Text("KEY")
	f.Space()
	f.Rune('(')
	for i, name := range node.Columns {
		name.Accept(f)
		if i < len(node.Columns)-1 {
			f.Rune(',')
			f.Space()
		}
	}
	f.Rune(')')
	f.Space()
	f.VisitForeignKeyClause(&node.FkClause)
}

func (f *SqliteGenerator) VisitForeignKeyClause(node *ast.ForeignKeyClause) {
	f.Text("REFERENCES")
	f.Space()
	node.ForeignTable.Accept(f)
	f.Space()

	if len(node.ForeignColumns) > 0 {
		f.Rune('(')
		for i, name := range node.ForeignColumns {
			name.Accept(f)
			if i < len(node.ForeignColumns)-1 {
				f.Rune(',')
				f.Space()
			}
		}
		f.Rune(')')
	}

	for _, action := range node.Actions {
		f.Space()
		action.Accept(f)
	}

	if node.Deferrable != nil {
		f.Space()
		if node.Deferrable.NotKeyword != nil {
			f.Text("NOT")
			f.Space()
		}

		f.Text("DEFERRABLE")
		f.Space()

		if node.Deferrable.InitiallyKeyword != nil {
			f.Text("INITIALLY")
			f.Space()
			f.Text(node.Deferrable.Deferrable.Text)
		}
	}
}

func (f *SqliteGenerator) VisitForeignKeyUpdateAction(node *ast.ForeignKeyUpdateAction) {
	f.Text("ON")
	f.Space()
	f.Text("UPDATE")
	f.Space()
	node.Action.Accept(f)
}

func (f *SqliteGenerator) VisitForeignKeyDeleteAction(node *ast.ForeignKeyDeleteAction) {
	f.Text("ON")
	f.Space()
	f.Text("DELETE")
	f.Space()
	node.Action.Accept(f)
}

func (f *SqliteGenerator) VisitForeignKeyActionNoAction(node *ast.NoAction) {
	f.Text("NO")
	f.Space()
	f.Text("ACTION")
}

func (f *SqliteGenerator) VisitLiteralSignedInteger(node *ast.LiteralSignedInteger) {
	f.Text(node.Token.Text)
}
