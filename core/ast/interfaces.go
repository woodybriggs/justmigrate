package ast

type Statement interface {
	Equalable[Statement]
	Accept(Visitor)
	nodeStatement()
}

func (node *Pragma) nodeStatement()            {}
func (node *BeginTransaction) nodeStatement()  {}
func (node *CommitTransaction) nodeStatement() {}
func (node *Select) nodeStatement()            {}
func (node *CreateTable) nodeStatement()       {}
func (node *AlterTable) nodeStatement()        {}
func (node *DropTable) nodeStatement()         {}
func (node *CreateTrigger) nodeStatement()     {}

type TableAlteration interface {
	Equalable[TableAlteration]
	Accept(Visitor)
	tableAlteration()
}

func (node *AddColumn) tableAlteration()  {}
func (node *DropColumn) tableAlteration() {}

// func (node *CreateView) nodeStatement()        {}

type TableConstraint interface {
	Equalable[TableConstraint]
	Accept(Visitor)
	nodeTableConstraint()
}

func (node *TableConstraint_Check) nodeTableConstraint()      {}
func (node *TableConstraint_PrimaryKey) nodeTableConstraint() {}
func (node *TableConstraint_ForeignKey) nodeTableConstraint() {}

type ColumnConstraint interface {
	Equalable[ColumnConstraint]
	Accept(Visitor)
	nodeColumnConstraint()
}

func (node *ColumnConstraint_PrimaryKey) nodeColumnConstraint() {}
func (node *ColumnConstraint_Default) nodeColumnConstraint()    {}
func (node *ColumnConstraint_NotNull) nodeColumnConstraint()    {}
func (node *ColumnConstraint_Generated) nodeColumnConstraint()  {}
func (node *ColumnConstraint_Check) nodeColumnConstraint()      {}
func (node *ColumnConstraint_Unique) nodeColumnConstraint()     {}
func (node *ColumnConstraint_Collate) nodeColumnConstraint()    {}

type ForeignKeyAction interface {
	Equalable[ForeignKeyAction]
	Accept(Visitor)
	nodeForeignKeyAction()
}

func (node *ForeignKeyDeleteAction) nodeForeignKeyAction() {}
func (node *ForeignKeyUpdateAction) nodeForeignKeyAction() {}

type ForeignKeyActionDo interface {
	Equalable[ForeignKeyActionDo]
	Accept(Visitor)
	nodeForeignKeyActionDo()
}

func (node *NoAction) nodeForeignKeyActionDo()   {}
func (node *Restrict) nodeForeignKeyActionDo()   {}
func (node *SetNull) nodeForeignKeyActionDo()    {}
func (node *SetDefault) nodeForeignKeyActionDo() {}
func (node *Cascade) nodeForeignKeyActionDo()    {}

type Expr interface {
	Equalable[Expr]
	Accept(Visitor)
	nodeExpression()
}

func (node ExprList) nodeExpression()                {}
func (node *BinaryOp) nodeExpression()               {}
func (node *UnaryOp) nodeExpression()                {}
func (node *FunctionCall) nodeExpression()           {}
func (node *ColumnName) nodeExpression()             {}
func (node *CaseExpression) nodeExpression()         {}
func (node *LiteralBoolean) nodeExpression()         {}
func (node *LiteralFloat) nodeExpression()           {}
func (node *LiteralSignedInteger) nodeExpression()   {}
func (node *LiteralUnsignedInteger) nodeExpression() {}
func (node *LiteralString) nodeExpression()          {}
func (node *LiteralNull) nodeExpression()            {}

type NumericLiteral interface {
	nodeNumericLiteral()
	Accept(Visitor)
}

func (node *LiteralFloat) nodeNumericLiteral()           {}
func (node *LiteralSignedInteger) nodeNumericLiteral()   {}
func (node *LiteralUnsignedInteger) nodeNumericLiteral() {}
