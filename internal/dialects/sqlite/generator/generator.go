package generator

import (
	"justmigrate/internal/backend/formatter"
	"justmigrate/internal/frontend/ast"
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

func (f *SqliteFormatter) VisitDropTable(node *ast.DropTable) {
	f.Group(func() {
		f.Keyword("DROP")
		f.Space()
		f.Keyword("TABLE")
		f.Space()

		if node.IfExists != nil {
			f.Keyword("IF")
			f.Space()
			f.Keyword("EXISTS")
			f.Space()
		}

		node.TableIdentifier.Accept(f)
	})
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

func (f *SqliteFormatter) VisitColumnConstraintCheck(node *ast.ColumnConstraint_Check) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}
	f.Keyword("CHECK")
	f.Space()
	f.Rune('(')
	node.Expr.Accept(f)
	f.Rune(')')
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
		f.Space()
	}

	f.VisitForeignKeyClause(&node.FkClause)
}

func (f *SqliteFormatter) VisitTableConstraintCheck(node *ast.TableConstraint_Check) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
		f.Space()
	}

	f.Keyword("CHECK")
	f.Space()
	f.Rune('(')
	node.Expr.Accept(f)
	f.Rune(')')
}

func (f *SqliteFormatter) VisitTableConstraintUnique(node *ast.TableConstraint_Unique) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
	}

	f.Keyword("UNIQUE")
	f.Space()

	f.Rune('(')
	for i, indexedCol := range node.IndexedColumns {

		f.VisitIndexedColumn(&indexedCol)

		if i != len(node.IndexedColumns)-1 {
			f.Rune(',')
			f.Space()
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

func (f *SqliteFormatter) VisitBinaryOp(node *ast.BinaryOp) {
	node.Lhs.Accept(f)
	f.Space()
	f.Text(node.Operator.Text)
	f.Space()

	if _, ok := node.Rhs.(ast.ExprList); ok {
		f.Rune('(')
		node.Rhs.Accept(f)
		f.Rune(')')
	} else {
		node.Rhs.Accept(f)
	}
}

func (f *SqliteFormatter) VisitColumnConstraintDefault(node *ast.ColumnConstraint_Default) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
		f.Space()
	}

	f.Keyword("DEFAULT")
	f.Space()

	// @TODO(woody): we should know already if the expr needs to be wrapped in parens
	switch node.Default.(type) {
	case *ast.FunctionCall:
		f.Rune('(')
		node.Default.Accept(f)
		f.Rune(')')
	default:
		node.Default.Accept(f)
	}

}

func (f *SqliteFormatter) VisitLiteralString(node *ast.LiteralString) {
	f.Rune('\'')
	f.Text(node.Value)
	f.Rune('\'')
}

func (f *SqliteFormatter) VisitColumnConstraintUnique(node *ast.ColumnConstraint_Unique) {
	if node.Name != nil {
		f.Keyword("CONSTRAINT")
		f.Space()
		node.Name.Name.Accept(f)
		f.Space()
	}

	f.Keyword("UNIQUE")

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

func (f *SqliteFormatter) VisitTableAlterationRenameTable(node *ast.RenameTable) {
	f.Keyword("RENAME")
	f.Space()
	f.Keyword("TO")
	f.Space()
	node.NewTableName.Accept(f)
}

func (f *SqliteFormatter) VisitInsertInto(node *ast.InsertInto) {
	f.Group(func() {
		f.Keyword("INSERT")
		f.Space()
		f.Keyword("INTO")
		f.Space()

		if node.Or != nil {
			// node.Or.Accept(f)
			f.Space()
		}

		node.CatalogObject.Accept(f)
		f.Space()

		if len(node.Columns) > 0 {
			f.Rune('(')
			f.Line()
			f.Indent(func() {
				for i, name := range node.Columns {
					name.Accept(f)
					if i < len(node.Columns)-1 {
						f.Rune(',')
						f.Line()
					}
				}
			})
			f.Line()
			f.Rune(')')
		}
		f.Space()

		node.Values.Accept(f)

		// @TODO(woody): we don't have returning clause in InsertInto statements yet
		// if node.ReturningClause != nil {
		// 	node.ReturningClause.Accept(f)
		// }
	})
}

func (f *SqliteFormatter) VisitInsertIntoValuesSelect(node *ast.InsertIntoValuesSelect) {
	node.Select.Accept(f)
	for i := range len(node.UpsertClauses) {
		node.UpsertClauses[i].Accept(f)
		if i < len(node.UpsertClauses)-1 {
			f.Space()
		}
	}
}

func (f *SqliteFormatter) VisitSelectFromTable(node *ast.SelectFromTable) {
	f.Keyword("SELECT")
	f.Space()

	f.Indent(func() {
		for i, resCol := range node.ResultsColumn {
			resCol.Accept(f)

			if i < len(node.ResultsColumn)-1 {
				f.Rune(',')
				f.Line()
			}
		}
	})
	f.Line()
	f.Keyword("FROM")
	f.Space()
	node.From.Accept(f)
}

func (f *SqliteFormatter) VisitUpsertClause(node *ast.UpsertClause) {
	f.Keyword("ON")
	f.Space()
	f.Keyword("CONFLICT")
	f.Space()

	f.Keyword("DO")
	f.Space()

	panic("not implemented")
}

func (f *SqliteFormatter) VisitLiteralNull(node *ast.LiteralNull) {
	f.Keyword("NULL")
}

func (f *SqliteFormatter) VisitFunctionCall(node *ast.FunctionCall) {
	f.Text(node.Name.Text)
	f.Rune('(')
	node.Args.Accept(f)
	f.Rune(')')
}

func (f *SqliteFormatter) VisitExprList(node ast.ExprList) {
	for i, expr := range node {
		expr.Accept(f)
		if i < len(node)-1 {
			f.Rune(',')
			f.Space()
		}
	}
}
