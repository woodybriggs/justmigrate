package generator

import (
	"woodybriggs/justmigrate/backend/formatter"
	"woodybriggs/justmigrate/frontend/ast"
)

type SqliteFormatter struct {
	formatter.Formatter
	ast.BaseVisitor
}

func NewSqliteFormatter(debug bool, formatter formatter.Formatter) *SqliteFormatter {
	return &SqliteFormatter{
		BaseVisitor: ast.BaseVisitor{
			Debug: debug,
		},
		Formatter: formatter,
	}
}

func (f *SqliteFormatter) VisitParseError(err *ast.ParseError) {}

func (f *SqliteFormatter) Keyword(keyword string) {
	f.Text(keyword)
}

func (f *SqliteFormatter) VisitStatements(node []ast.Statement) {
	for _, stmt := range node {
		stmt.Accept(f)
		f.Rune(';')
		f.Break()
		f.Break()
	}
}

func (f *SqliteFormatter) VisitCreateTable(node *ast.CreateTable) {
	f.Group(func() {
		f.Keyword("CREATE")
		f.Space()
		f.Keyword("TABLE")
		f.Space()

		if node.IfNotExist != nil {
			f.Keyword("IF")
			f.Space()
			f.Keyword("NOT")
			f.Space()
			f.Keyword("EXISTS")
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

func (f *SqliteFormatter) VisitAlterTable(node *ast.AlterTable) {
	f.Group(func() {
		f.Keyword("ALTER")
		f.Space()
		f.Keyword("TABLE")
		f.Space()
		node.TableIdentifier.Accept(f)
		f.Space()
		node.Alteration.Accept(f)
	})
}

func (f *SqliteFormatter) VisitTableAlterationAddColumn(node *ast.AddColumn) {
	f.Keyword("ADD")
	f.Space()
	f.Keyword("COLUMN")
	f.Space()
	node.ColumnDefinition.Accept(f)
}

func (f *SqliteFormatter) VisitTableAlterationDropColumn(node *ast.DropColumn) {
	f.Keyword("DROP")
	f.Space()
	f.Keyword("COLUMN")
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

func (f *SqliteFormatter) VisitColumnConstraintNotNull(node *ast.ColumnConstraint_NotNull) {
	f.Keyword("NOT")
	f.Space()
	f.Keyword("NULL")
}

func (f *SqliteFormatter) VisitColumnConstraintPrimaryKey(node *ast.ColumnConstraint_PrimaryKey) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Keyword("PRIMARY")
	f.Space()
	f.Keyword("KEY")

	if node.Order != nil {
		f.Space()
		f.Text(node.Order.Text)
		f.Space()
	}

	if node.ConflictClause != nil {
		f.Space()
		f.Keyword("ON")
		f.Space()
		f.Keyword("CONFLICT")
		f.Space()
		f.Text(node.ConflictClause.Action.Text)
		f.Space()
	}

	if node.AutoIncrement != nil {
		f.Space()
		f.Keyword("AUTOINCREMENT")
		f.Space()
	}
}

func (f *SqliteFormatter) VisitColumnConstraintForeignKey(node *ast.ColumnConstraint_ForeignKey) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.VisitForeignKeyClause(&node.FkClause)
}

func (f *SqliteFormatter) VisitTableConstraintPrimaryKey(node *ast.TableConstraint_PrimaryKey) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Keyword("PRIMARY")
	f.Space()
	f.Keyword("KEY")
	f.Space()

	f.Rune('(')
	if node.AutoIncrement != nil {
		f.VisitIndexedColumn(&node.IndexedColumns[0])
		f.Space()
		f.Keyword("AUTOINCREMENT")
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
		f.Keyword("ON")
		f.Space()
		f.Keyword("CONFLICT")
		f.Space()
		f.Keyword(node.ConflictClause.Action.Text)
		f.Space()
	}
}

func (f *SqliteFormatter) VisitIndexedColumn(node *ast.IndexedColumn) {
	node.Subject.Accept(f)

	if node.Collation != nil {
		f.Space()
		f.Keyword("COLLATE")
		f.Space()
		node.Collation.Name.Accept(f)
	}

	if node.Order != nil {
		f.Space()
		f.Text(node.Order.Text)
	}
}

func (f *SqliteFormatter) VisitTableConstraintForeignKey(node *ast.TableConstraint_ForeignKey) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Keyword("FOREIGN")
	f.Space()
	f.Keyword("KEY")
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

func (f *SqliteFormatter) VisitForeignKeyClause(node *ast.ForeignKeyClause) {
	f.Keyword("REFERENCES")
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
			f.Keyword("NOT")
			f.Space()
		}

		f.Keyword("DEFERRABLE")
		f.Space()

		if node.Deferrable.InitiallyKeyword != nil {
			f.Keyword("INITIALLY")
			f.Space()
			f.Text(node.Deferrable.Deferrable.Text)
		}
	}
}

func (f *SqliteFormatter) VisitForeignKeyUpdateAction(node *ast.ForeignKeyUpdateAction) {
	f.Keyword("ON")
	f.Space()
	f.Keyword("UPDATE")
	f.Space()
	node.Action.Accept(f)
}

func (f *SqliteFormatter) VisitForeignKeyDeleteAction(node *ast.ForeignKeyDeleteAction) {
	f.Keyword("ON")
	f.Space()
	f.Keyword("DELETE")
	f.Space()
	node.Action.Accept(f)
}

func (f *SqliteFormatter) VisitForeignKeyActionNoAction(node *ast.NoAction) {
	f.Keyword("NO")
	f.Space()
	f.Keyword("ACTION")
}

func (f *SqliteFormatter) VisitLiteralSignedInteger(node *ast.LiteralSignedInteger) {
	f.Text(node.Token.Text)
}
