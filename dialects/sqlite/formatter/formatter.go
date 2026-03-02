package formatter

import (
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/formatter"
)

type SqliteFormatter struct {
	formatter.Formatter
}

func (f *SqliteFormatter) VisitStatements(node []ast.Statement) {
	for _, stmt := range node {
		stmt.Accept(f)
		f.Rune(';')
		f.Break()
	}
}

func (f *SqliteFormatter) VisitCreateIndex(node *ast.CreateIndex) {}

func (f *SqliteFormatter) VisitCreateTable(node *ast.CreateTable) {}

func (f *SqliteFormatter) VisitAlterTable(node *ast.AlterTable) {}

func (f *SqliteFormatter) VisitColumnConstraintCheck(node *ast.ColumnConstraint_Check) {}

func (f *SqliteFormatter) VisitColumnConstraintPrimaryKey(node *ast.ColumnConstraint_PrimaryKey) {}

func (f *SqliteFormatter) VisitDropTable(node *ast.DropTable) {}

func (f *SqliteFormatter) VisitCreateView(node *ast.CreateView) {}

func (f *SqliteFormatter) VisitTableConstraintCheck(node *ast.TableConstraint_Check) {}

func (f *SqliteFormatter) VisitTableConstraintPrimaryKey(node *ast.TableConstraint_PrimaryKey) {}

func (f *SqliteFormatter) VisitTableConstraintForeignKey(node *ast.TableConstraint_ForeignKey) {}

func (f *SqliteFormatter) VisitColumnConstraintDefault(node *ast.ColumnConstraint_Default) {}

func (f *SqliteFormatter) VisitColumnConstraintUnique(node *ast.ColumnConstraint_Unique) {}

func (f *SqliteFormatter) VisitColumnConstraintGenerated(node *ast.ColumnConstraint_Generated) {}

func (f *SqliteFormatter) VisitColumnConstraintCollate(node *ast.ColumnConstraint_Collate) {}

func (f *SqliteFormatter) VisitColumnConstraintNotNull(node *ast.ColumnConstraint_NotNull) {}

func (f *SqliteFormatter) VisitForeignKeyDeleteAction(node *ast.ForeignKeyDeleteAction) {}

func (f *SqliteFormatter) VisitForeignKeyUpdateAction(node *ast.ForeignKeyUpdateAction) {}

func (f *SqliteFormatter) VisitIdentifier(node *ast.Identifier) {}

func (f *SqliteFormatter) VisitExprList(node ast.ExprList) {}

func (f *SqliteFormatter) VisitLiteralString(node *ast.LiteralString) {}

func (f *SqliteFormatter) VisitLiteralBoolean(node *ast.LiteralBoolean) {}

func (f *SqliteFormatter) VisitLiteralSignedInteger(node *ast.LiteralSignedInteger) {}

func (f *SqliteFormatter) VisitLiteralUnsignedInteger(node *ast.LiteralUnsignedInteger) {}

func (f *SqliteFormatter) VisitLiteralFloat(node *ast.LiteralFloat) {}

func (f *SqliteFormatter) VisitLiteralNull(node *ast.LiteralNull) {}

func (f *SqliteFormatter) VisitFunctionCall(node *ast.FunctionCall) {}

func (f *SqliteFormatter) VisitColumnName(node *ast.ColumnName) {}

func (f *SqliteFormatter) VisitBinaryOp(node *ast.BinaryOp) {}

func (f *SqliteFormatter) VisitCaseExpression(node *ast.CaseExpression) {}

// func ToSQL(f formatter.Formatter, node []ast.Statement) {

// }

// func statementToSQL(f formatter.Formatter, node ast.Statement) {
// 	switch typ := node.(type) {
// 	case *ast.CreateTable:
// 		{
// 			createTableToSQL(f, typ)
// 		}
// 	default:
// 		panic(ErrNotImplemented)
// 	}
// }

// func createTableToSQL(f formatter.Formatter, node *ast.CreateTable) {
// 	f.Group(func() {
// 		f.Text(node.CreateKeyword.Text)
// 		if node.Temporary != nil {
// 			f.Space()
// 			f.Text(node.Temporary.Text)
// 		}

// 		f.Space()
// 		f.Text(node.TableKeyword.Text)

// 		if node.IfNotExist != nil {
// 			f.Space()
// 			ifNotExistsToSQL(f, node.IfNotExist)
// 		}

// 		f.Space()
// 		node.TableIdentifier.ToSql(f)
// 		f.Space()

// 		f.Rune('(')
// 		f.Break()

// 		f.Indent(func() {
// 			node.TableDefinition.ToSql(f)
// 		})
// 		f.Break()
// 		f.Rune(')')
// 		f.Space()
// 		if node.TableOptions != nil {
// 			node.TableOptions.ToSql(f)
// 		}
// 	})
// }

// func ifNotExistsToSQL(f formatter.Formatter, node *ast.IfNotExists) {
// 	f.Text(node.If.Text)
// 	f.Space()
// 	f.Text(node.Not.Text)
// 	f.Space()
// 	f.Text(node.Exists.Text)
// }
