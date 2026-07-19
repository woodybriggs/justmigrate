package ast

type Statement interface {
	Equalable
	Accept(Visitor)
	nodeStatement()
}

func (*Pragma) nodeStatement()            {}
func (*BeginTransaction) nodeStatement()  {}
func (*CommitTransaction) nodeStatement() {}
func (*SelectFromTable) nodeStatement()   {}
func (*CreateTable) nodeStatement()       {}
func (*AlterTable) nodeStatement()        {}
func (*DropTable) nodeStatement()         {}
func (*CreateTrigger) nodeStatement()     {}
func (*InsertInto) nodeStatement()        {}

type TableAlteration interface {
	Equalable
	Accept(Visitor)
	tableAlteration()
}

func (*AddColumn) tableAlteration()   {}
func (*DropColumn) tableAlteration()  {}
func (*RenameTable) tableAlteration() {}

// func (node *CreateView) nodeStatement()        {}

type TableConstraint interface {
	Equalable
	Accept(Visitor)
	nodeTableConstraint()
	GetConstraintName() *Identifier
}

func (node *TableConstraint_Check) nodeTableConstraint()      {}
func (node *TableConstraint_PrimaryKey) nodeTableConstraint() {}
func (node *TableConstraint_ForeignKey) nodeTableConstraint() {}
func (node *TableConstraint_Unique) nodeTableConstraint()     {}
func (node *ParseError) nodeTableConstraint()                 {}

func (node *TableConstraint_Check) GetConstraintName() *Identifier {
	if node.Name != nil {
		return &node.Name.Name
	}
	return nil
}

func (node *TableConstraint_PrimaryKey) GetConstraintName() *Identifier {
	if node.Name != nil {
		return &node.Name.Name
	}
	return nil
}

func (node *TableConstraint_ForeignKey) GetConstraintName() *Identifier {
	if node.Name != nil {
		return &node.Name.Name
	}
	return nil
}

func (node *TableConstraint_Unique) GetConstraintName() *Identifier {
	if node.Name != nil {
		return &node.Name.Name
	}
	return nil
}

type ColumnConstraint interface {
	Equalable
	Accept(Visitor)
	nodeColumnConstraint()
}

func (node *ColumnConstraint_PrimaryKey) nodeColumnConstraint() {}
func (node *ColumnConstraint_ForeignKey) nodeColumnConstraint() {}
func (node *ColumnConstraint_Default) nodeColumnConstraint()    {}
func (node *ColumnConstraint_NotNull) nodeColumnConstraint()    {}
func (node *ColumnConstraint_Generated) nodeColumnConstraint()  {}
func (node *ColumnConstraint_Check) nodeColumnConstraint()      {}
func (node *ColumnConstraint_Unique) nodeColumnConstraint()     {}
func (node *ColumnConstraint_Collate) nodeColumnConstraint()    {}
func (node *ParseError) nodeColumnConstraint()                  {}

type ForeignKeyAction interface {
	Equalable
	Accept(Visitor)
	nodeForeignKeyAction()
}

func (node *ForeignKeyDeleteAction) nodeForeignKeyAction() {}
func (node *ForeignKeyUpdateAction) nodeForeignKeyAction() {}

type ForeignKeyActionDo interface {
	Equalable
	Accept(Visitor)
	nodeForeignKeyActionDo()
}

func (node *NoAction) nodeForeignKeyActionDo()   {}
func (node *Restrict) nodeForeignKeyActionDo()   {}
func (node *SetNull) nodeForeignKeyActionDo()    {}
func (node *SetDefault) nodeForeignKeyActionDo() {}
func (node *Cascade) nodeForeignKeyActionDo()    {}

type Expr interface {
	Equalable
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
func (node *ParseError) nodeExpression()             {}

type NumericLiteral interface {
	Equalable
	nodeNumericLiteral()
	Accept(Visitor)
}

func (node *LiteralFloat) nodeNumericLiteral()           {}
func (node *LiteralSignedInteger) nodeNumericLiteral()   {}
func (node *LiteralUnsignedInteger) nodeNumericLiteral() {}

type InsertIntoValues interface {
	Equalable
	nodeInsertIntoValues()
	Accept(Visitor)
}

func (*InsertIntoValuesSelect) nodeInsertIntoValues()   {}
func (*InsertIntoValuesExprs) nodeInsertIntoValues()    {}
func (*InsertIntoValuesDefaults) nodeInsertIntoValues() {}
