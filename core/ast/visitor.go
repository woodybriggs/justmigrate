package ast

import (
	"fmt"
	"os"
)

type Visitor interface {
	VisitParseError(*ParseError)

	VisitDropTable(*DropTable)

	VisitCreateTable(*CreateTable)
	VisitCreateIndex(*CreateIndex)
	VisitCreateView(*CreateView)
	VisitAlterTable(*AlterTable)

	VisitTableAlterationAddColumn(*AddColumn)
	VisitTableAlterationDropColumn(*DropColumn)

	VisitTableConstraintCheck(*TableConstraint_Check)
	VisitTableConstraintPrimaryKey(*TableConstraint_PrimaryKey)
	VisitTableConstraintForeignKey(*TableConstraint_ForeignKey)

	VisitColumnConstraintPrimaryKey(*ColumnConstraint_PrimaryKey)
	VisitColumnConstraintForeignKey(*ColumnConstraint_ForeignKey)
	VisitColumnConstraintDefault(*ColumnConstraint_Default)
	VisitColumnConstraintCheck(*ColumnConstraint_Check)
	VisitColumnConstraintUnique(*ColumnConstraint_Unique)
	VisitColumnConstraintGenerated(*ColumnConstraint_Generated)
	VisitColumnConstraintCollate(*ColumnConstraint_Collate)
	VisitColumnConstraintNotNull(*ColumnConstraint_NotNull)

	VisitForeignKeyDeleteAction(*ForeignKeyDeleteAction)
	VisitForeignKeyUpdateAction(*ForeignKeyUpdateAction)

	VisitForeignKeyActionNoAction(*NoAction)
	VisitForeignKeyActionCascade(*Cascade)
	VisitForeignKeyActionSetNull(*SetNull)
	VisitForeignKeyActionSetDefault(*SetDefault)
	VisitForeignKeyActionRestrict(*Restrict)

	VisitIdentifier(*Identifier)
	VisitExprList(ExprList)
	VisitLiteralString(*LiteralString)
	VisitLiteralBoolean(*LiteralBoolean)
	VisitLiteralSignedInteger(*LiteralSignedInteger)
	VisitLiteralUnsignedInteger(*LiteralUnsignedInteger)
	VisitLiteralFloat(*LiteralFloat)
	VisitLiteralNull(*LiteralNull)

	VisitFunctionCall(*FunctionCall)
	VisitColumnName(*ColumnName)
	VisitBinaryOp(*BinaryOp)
	VisitCaseExpression(*CaseExpression)

	VisitColumnDefinition(*ColumnDefinition)
	VisitTypeName(*TypeName)
	VisitCatalogObjectIdentifier(*CatalogObjectIdentifier)
}

func (node *ParseError) Accept(v Visitor) {
	v.VisitParseError(node)
}

func (node *TypeName) Accept(v Visitor) {
	v.VisitTypeName(node)
}

func (node *AddColumn) Accept(v Visitor) {
	v.VisitTableAlterationAddColumn(node)
}

func (node *DropColumn) Accept(v Visitor) {
	v.VisitTableAlterationDropColumn(node)
}

func (node *DropTable) Accept(v Visitor) {
	v.VisitDropTable(node)
}

func (node *CreateTable) Accept(v Visitor) {
	v.VisitCreateTable(node)
}

func (node *CreateView) Accept(v Visitor) {
	v.VisitCreateView(node)
}

func (node *CreateIndex) Accept(v Visitor) {
	v.VisitCreateIndex(node)
}

func (node *AlterTable) Accept(v Visitor) {
	v.VisitAlterTable(node)
}

func (node *TableConstraint_Check) Accept(v Visitor) {
	v.VisitTableConstraintCheck(node)
}

func (node *TableConstraint_PrimaryKey) Accept(v Visitor) {
	v.VisitTableConstraintPrimaryKey(node)
}

func (node *TableConstraint_ForeignKey) Accept(v Visitor) {
	v.VisitTableConstraintForeignKey(node)
}

func (node *ColumnConstraint_PrimaryKey) Accept(v Visitor) {
	v.VisitColumnConstraintPrimaryKey(node)
}

func (node *ColumnConstraint_Default) Accept(v Visitor) {
	v.VisitColumnConstraintDefault(node)
}

func (node *ColumnConstraint_Check) Accept(v Visitor) {
	v.VisitColumnConstraintCheck(node)
}

func (node *ColumnConstraint_Unique) Accept(v Visitor) {
	v.VisitColumnConstraintUnique(node)
}

func (node *ColumnConstraint_Generated) Accept(v Visitor) {
	v.VisitColumnConstraintGenerated(node)
}

func (node *ColumnConstraint_Collate) Accept(v Visitor) {
	v.VisitColumnConstraintCollate(node)
}

func (node *ColumnConstraint_NotNull) Accept(v Visitor) {
	v.VisitColumnConstraintNotNull(node)
}

func (node *ForeignKeyDeleteAction) Accept(v Visitor) {
	v.VisitForeignKeyDeleteAction(node)
}

func (node *ForeignKeyUpdateAction) Accept(v Visitor) {
	v.VisitForeignKeyUpdateAction(node)
}

func (node ExprList) Accept(v Visitor) {
	v.VisitExprList(node)
}

func (node *LiteralBoolean) Accept(v Visitor) {
	v.VisitLiteralBoolean(node)
}

func (node *LiteralFloat) Accept(v Visitor) {
	v.VisitLiteralFloat(node)
}

func (node *LiteralNull) Accept(v Visitor) {
	v.VisitLiteralNull(node)
}

func (node *LiteralSignedInteger) Accept(v Visitor) {
	v.VisitLiteralSignedInteger(node)
}

func (node *LiteralUnsignedInteger) Accept(v Visitor) {
	v.VisitLiteralUnsignedInteger(node)
}

func (node *LiteralString) Accept(v Visitor) {
	v.VisitLiteralString(node)
}

func (node *Identifier) Accept(v Visitor) {
	v.VisitIdentifier(node)
}

func (node *FunctionCall) Accept(v Visitor) {
	v.VisitFunctionCall(node)
}

func (node *ColumnName) Accept(v Visitor) {
	v.VisitColumnName(node)
}

func (node *BinaryOp) Accept(v Visitor) {
	v.VisitBinaryOp(node)
}

func (node *CaseExpression) Accept(v Visitor) {
	v.VisitCaseExpression(node)
}

func (node *ColumnDefinition) Accept(v Visitor) {
	v.VisitColumnDefinition(node)
}

func (node *CatalogObjectIdentifier) Accept(v Visitor) {
	v.VisitCatalogObjectIdentifier(node)
}

func (node *NoAction) Accept(v Visitor) {
	v.VisitForeignKeyActionNoAction(node)
}

func (node *Cascade) Accept(v Visitor) {
	v.VisitForeignKeyActionCascade(node)
}

func (node *Restrict) Accept(v Visitor) {
	v.VisitForeignKeyActionRestrict(node)
}

func (node *SetDefault) Accept(v Visitor) {
	v.VisitForeignKeyActionSetDefault(node)
}

func (node *SetNull) Accept(v Visitor) {
	v.VisitForeignKeyActionSetNull(node)
}

func (node *ColumnConstraint_ForeignKey) Accept(v Visitor) {
	v.VisitColumnConstraintForeignKey(node)
}

type BaseVisitor struct {
	Debug bool
}

func (v *BaseVisitor) VisitForeignKeyActionNoAction(*NoAction) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyActionNoAction")
	}
}
func (v *BaseVisitor) VisitForeignKeyActionCascade(*Cascade) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyActionCascade")
	}
}
func (v *BaseVisitor) VisitForeignKeyActionSetNull(*SetNull) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyActionSetNull")
	}
}
func (v *BaseVisitor) VisitForeignKeyActionSetDefault(*SetDefault) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyActionSetDefault")
	}
}
func (v *BaseVisitor) VisitForeignKeyActionRestrict(*Restrict) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyActionRestrict")
	}
}

func (v *BaseVisitor) VisitTypeName(*TypeName) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTypeName")
	}
}

func (v *BaseVisitor) VisitDropTable(*DropTable) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitDropTable")
	}
}
func (v *BaseVisitor) VisitCreateTable(*CreateTable) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitCreateTable")
	}
}
func (v *BaseVisitor) VisitCreateIndex(*CreateIndex) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitCreateIndex")
	}
}
func (v *BaseVisitor) VisitCreateView(*CreateView) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitCreateView")
	}
}
func (v *BaseVisitor) VisitAlterTable(*AlterTable) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitAlterTable")
	}
}
func (v *BaseVisitor) VisitTableAlterationAddColumn(*AddColumn) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTableAlterationAddColumn")
	}
}
func (v *BaseVisitor) VisitTableAlterationDropColumn(*DropColumn) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTableAlterationDropColumn")
	}
}
func (v *BaseVisitor) VisitTableConstraintCheck(*TableConstraint_Check) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTableConstraintCheck")
	}
}
func (v *BaseVisitor) VisitTableConstraintPrimaryKey(*TableConstraint_PrimaryKey) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTableConstraintPrimaryKey")
	}
}
func (v *BaseVisitor) VisitTableConstraintForeignKey(*TableConstraint_ForeignKey) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitTableConstraintForeignKey")
	}
}
func (v *BaseVisitor) VisitColumnConstraintPrimaryKey(*ColumnConstraint_PrimaryKey) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintPrimaryKey")
	}
}
func (v *BaseVisitor) VisitColumnConstraintForeignKey(*ColumnConstraint_ForeignKey) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintForeignKey")
	}
}
func (v *BaseVisitor) VisitColumnConstraintDefault(*ColumnConstraint_Default) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintDefault")
	}
}
func (v *BaseVisitor) VisitColumnConstraintCheck(*ColumnConstraint_Check) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintCheck")
	}
}
func (v *BaseVisitor) VisitColumnConstraintUnique(*ColumnConstraint_Unique) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintUnique")
	}
}
func (v *BaseVisitor) VisitColumnConstraintGenerated(*ColumnConstraint_Generated) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintGenerated")
	}
}
func (v *BaseVisitor) VisitColumnConstraintCollate(*ColumnConstraint_Collate) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintCollate")
	}
}
func (v *BaseVisitor) VisitColumnConstraintNotNull(*ColumnConstraint_NotNull) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnConstraintNotNull")
	}
}
func (v *BaseVisitor) VisitForeignKeyDeleteAction(*ForeignKeyDeleteAction) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyDeleteAction")
	}
}
func (v *BaseVisitor) VisitForeignKeyUpdateAction(*ForeignKeyUpdateAction) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitForeignKeyUpdateAction")
	}
}
func (v *BaseVisitor) VisitIdentifier(*Identifier) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitIdentifier")
	}
}
func (v *BaseVisitor) VisitExprList(ExprList) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitExprList")
	}
}
func (v *BaseVisitor) VisitLiteralString(*LiteralString) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralString")
	}
}
func (v *BaseVisitor) VisitLiteralBoolean(*LiteralBoolean) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralBoolean")
	}
}
func (v *BaseVisitor) VisitLiteralSignedInteger(*LiteralSignedInteger) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralSignedInteger")
	}
}
func (v *BaseVisitor) VisitLiteralUnsignedInteger(*LiteralUnsignedInteger) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralUnsignedInteger")
	}
}
func (v *BaseVisitor) VisitLiteralFloat(*LiteralFloat) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralFloat")
	}
}
func (v *BaseVisitor) VisitLiteralNull(*LiteralNull) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitLiteralNull")
	}
}
func (v *BaseVisitor) VisitFunctionCall(*FunctionCall) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitFunctionCall")
	}
}
func (v *BaseVisitor) VisitColumnName(*ColumnName) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnName")
	}
}
func (v *BaseVisitor) VisitBinaryOp(*BinaryOp) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitBinaryOp")
	}
}
func (v *BaseVisitor) VisitCaseExpression(*CaseExpression) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitCaseExpression")
	}
}
func (v *BaseVisitor) VisitColumnDefinition(*ColumnDefinition) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitColumnDefinition")
	}
}
func (v *BaseVisitor) VisitCatalogObjectIdentifier(*CatalogObjectIdentifier) {
	if v.Debug {
		fmt.Fprintf(os.Stderr, "VisitCatalogObjectIdentifier")
	}
}
