package ast

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"woodybriggs/justmigrate/core/tik"
	"woodybriggs/justmigrate/formatter"
)

type AstNode interface {
	node()
}

type Statement interface {
	AstNode
	nodeStatement()
	ToSql(f formatter.Formatter)
}

type TableConstraint interface {
	AstNode
	nodeTableConstraint()
	Eq(other TableConstraint) bool
	ToSql(f formatter.Formatter)
}

type ColumnConstraint interface {
	AstNode
	nodeColumnConstraint()
	Eq(other ColumnConstraint) bool
	ToSql(f formatter.Formatter)
}

type Expr interface {
	AstNode
	nodeExpression()
	Eq(other Expr) bool
}

type Literal interface {
	AstNode
	Expr
	nodeLiteral()
}

type AstNodeList []AstNode

func (node AstNodeList) node() {}

type Identifier tik.Token

func (t *Identifier) node()            {}
func (t *Identifier) nodeExpression()  {}
func (t *Identifier) nodeIdentifier()  {}
func (t *Identifier) nodePragmaValue() {}

func (node *Identifier) Eq(other Expr) bool {
	if other == nil {
		return false
	}

	switch t := other.(type) {
	case *Identifier:
		return node.Text == t.Text
	case Expr:
		return node.AsExpr().Eq(t)
	default:
		return false
	}
}

func (node *Identifier) AsExpr() Expr {
	return Expr(node)
}

func (node *Identifier) ToSql(f formatter.Formatter) {
	f.Identifier(node.Text)
}

type Keyword tik.Token

func MakeKeyword(token tik.Token) *Keyword {
	result := Keyword(token)
	return &result
}

func (node *Keyword) ToSql(f formatter.Formatter) {
	f.Text(node.Text)
}

func (node *Keyword) node() {}

func (node *Keyword) Eq(other *Keyword) bool {

	if node == nil && other == nil {
		return true
	}

	if node != nil && other != nil {
		return node.Kind == other.Kind
	}

	return false
}

type IfExists struct {
	If     Keyword
	Exists Keyword
}

func (node *IfExists) ToSql(f formatter.Formatter) {
	f.Text(node.If.Text)
	f.Space()
	f.Text(node.Exists.Text)
}

type DropTable struct {
	IfExists        *IfExists
	TableIdentifier CatalogObjectIdentifier
}

func (node *DropTable) node()          {}
func (node *DropTable) nodeStatement() {}

func (node *DropTable) ToSql(f formatter.Formatter) {
	f.Text("DROP")
	f.Space()
	f.Text("TABLE")
	f.Space()
	if node.IfExists != nil {
		node.IfExists.ToSql(f)
		f.Space()
	}
	node.TableIdentifier.ToSql(f)
}

type AlterTable struct {
	AlterKeyword    Keyword
	TableKeyword    Keyword
	TableIdentifier *CatalogObjectIdentifier
	Alteration      TableAlteration
}

func (node *AlterTable) node()          {}
func (node *AlterTable) nodeStatement() {}
func (node *AlterTable) ToSql(f formatter.Formatter) {
	f.Group(func() {
		f.Text(node.AlterKeyword.Text)
		f.Space()
		f.Text(node.TableKeyword.Text)
		f.Line()
		f.Indent(func() {
			node.TableIdentifier.ToSql(f)
		})
		f.Line()
		node.Alteration.ToSql(f)
	})
}

type TableAlteration interface {
	AstNode
	tableAlteration()
	ToSql(f formatter.Formatter)
}

type AddColumn struct {
	AddKeyword       Keyword
	ColumnKeyword    *Keyword
	ColumnDefinition ColumnDefinition
}

func (node *AddColumn) ToSql(f formatter.Formatter) {
	f.Text("ADD")
	f.Space()
	f.Text("COLUMN")
	f.Line()
	f.Indent(func() {
		node.ColumnDefinition.ToSql(f)
	})
}

func (node *AddColumn) node()            {}
func (node *AddColumn) tableAlteration() {}

type DropColumn struct {
	DropKeyword   Keyword
	ColumnKeyword *Keyword
	ColumnName    Identifier
}

func (node *DropColumn) ToSql(f formatter.Formatter) {
	f.Text("DROP")
	f.Space()
	f.Text("COLUMN")
	f.Line()
	f.Indent(func() {
		node.ColumnName.ToSql(f)
	})
}

func (node *DropColumn) node()            {}
func (node *DropColumn) tableAlteration() {}

type Pragma struct {
	Name  CatalogObjectIdentifier
	Value PragmaValue
}

func (node *Pragma) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *Pragma) node()          {}
func (node *Pragma) nodeStatement() {}

type PragmaValue interface {
	AstNode
	nodePragmaValue()
}

type BeginTransaction struct{}

func (node *BeginTransaction) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *BeginTransaction) node()          {}
func (node *BeginTransaction) nodeStatement() {}

type CommitTransaction struct{}

func (node *CommitTransaction) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *CommitTransaction) node()          {}
func (node *CommitTransaction) nodeStatement() {}

type Select struct{}

func (node *Select) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *Select) node()          {}
func (node *Select) nodeStatement() {}

type CreateTable struct {
	CreateKeyword Keyword
	TableKeyword  Keyword
	Temporary     *Keyword
	IfNotExist    *IfNotExists

	TableIdentifier *CatalogObjectIdentifier
	TableDefinition *TableDefinition

	TableOptions *TableOptions
}

func MakeCreateTable(
	create Keyword,
	temporary *Keyword,
	table Keyword,
	ifNotExists *IfNotExists,
	tableIdent *CatalogObjectIdentifier,
	tableDefinition *TableDefinition,
	tableOptions *TableOptions,
) *CreateTable {
	return &CreateTable{
		CreateKeyword:   create,
		Temporary:       temporary,
		TableKeyword:    table,
		IfNotExist:      ifNotExists,
		TableIdentifier: tableIdent,
		TableDefinition: tableDefinition,
		TableOptions:    tableOptions,
	}
}

func (node *CreateTable) node()          {}
func (node *CreateTable) nodeStatement() {}

func (node *CreateTable) ToSql(f formatter.Formatter) {
	f.Group(func() {
		f.Text(node.CreateKeyword.Text)
		if node.Temporary != nil {
			f.Space()
			f.Text(node.Temporary.Text)
		}

		f.Space()
		f.Text(node.TableKeyword.Text)

		if node.IfNotExist != nil {
			f.Space()
			node.IfNotExist.ToSql(f)
		}

		f.Space()
		node.TableIdentifier.ToSql(f)
		f.Space()

		f.Rune('(')
		f.Break()

		f.Indent(func() {
			node.TableDefinition.ToSql(f)
		})
		f.Break()
		f.Rune(')')
		f.Space()
		if node.TableOptions != nil {
			node.TableOptions.ToSql(f)
		}
	})
}

type CreateVirtualTable struct {
	IfNotExist      AstNode
	TableIdentifier CatalogObjectIdentifier
	ModuleName      Identifier
	ModuleArgs      []string
}

func (node *CreateVirtualTable) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *CreateVirtualTable) node()          {}
func (node *CreateVirtualTable) nodeStatement() {}

type CreateIndex struct {
	CreateKeyword   Keyword
	IndexKeyword    Keyword
	Unique          AstNode
	IfNotExists     AstNode
	IndexIdentifier CatalogObjectIdentifier
	OnTable         CatalogObjectIdentifier
	IndexedColumns  []IndexedColumn
	WhereExpr       Expr
}

func (node *CreateIndex) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *CreateIndex) node()          {}
func (node *CreateIndex) nodeStatement() {}

type TriggerTime interface {
	AstNode
	triggerTime()
}

type TriggerEvent interface {
	AstNode
	triggerEvent()
}

type CreateTrigger struct {
	CreateKeyword     Keyword
	TriggerKeyword    Keyword
	Temporary         *Keyword
	IfNotExists       AstNode
	TriggerIdentifier CatalogObjectIdentifier
	TriggerTime       TriggerTime
	TriggerEvent      TriggerEvent
	OnTable           AstNode
}

func (node *CreateTrigger) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *CreateTrigger) node()          {}
func (node *CreateTrigger) nodeStatement() {}

type TriggerTimeBefore struct {
	BeforeKeyword Keyword
}

func (node *TriggerTimeBefore) node()        {}
func (node *TriggerTimeBefore) triggerTime() {}

type TriggerTimeAfter struct {
	AfterKeyword Keyword
}

func (node *TriggerTimeAfter) node()        {}
func (node *TriggerTimeAfter) triggerTime() {}

type TriggerTimeInsteadOf struct {
	InsteadKeyword Keyword
	Of             Keyword
}

func (node *TriggerTimeInsteadOf) node()        {}
func (node *TriggerTimeInsteadOf) triggerTime() {}

type TriggerEventDelete struct {
	DeleteKeyword Keyword
}

func (node *TriggerEventDelete) node()         {}
func (node *TriggerEventDelete) triggerEvent() {}

type TriggerEventInsert struct {
	InsertKeyword Keyword
}

func (node *TriggerEventInsert) node()         {}
func (node *TriggerEventInsert) triggerEvent() {}

type TriggerEventUpdate struct {
	UpdateKeyword Keyword
}

func (node *TriggerEventUpdate) node()         {}
func (node *TriggerEventUpdate) triggerEvent() {}

type TriggerEventUpdateOf struct {
	UpdateKeyword Keyword
	Of            Keyword
	Columns       AstNodeList
}

func (node *TriggerEventUpdateOf) node()         {}
func (node *TriggerEventUpdateOf) triggerEvent() {}

type IndexedColumn struct {
	Subject   Expr
	Collation *Collation
	Order     *Keyword
}

func (node *IndexedColumn) ToSql(f formatter.Formatter) {
	switch typ := node.Subject.(type) {
	case *Identifier:
		{
			typ.ToSql(f)
		}
	default:
		{
			panic("IndexedColumn.ToSql not implemented for type")
		}
	}
}

func (node *IndexedColumn) node() {}
func (node *IndexedColumn) Eq(other *IndexedColumn) bool {
	result := true

	result = result && node.Subject.Eq(other.Subject)

	if node.Collation != nil && other.Collation != nil {
		result = result && node.Collation.Name.Eq(&node.Collation.Name)
	} else if node.Collation.Name.Eq(other.Collation.Name.AsExpr()) {
		return false
	}

	if node.Order != nil && other.Order != nil {
		result = result && node.Order.Eq(other.Order)
	} else if node.Order != nil || other.Order != nil {
		return false
	}

	return result
}

type CreateView struct {
	IfNotExists    *IfNotExists
	ViewIdentifier CatalogObjectIdentifier
	Columns        []Identifier
	AsSelect       AstNode
}

func (node *CreateView) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *CreateView) node()          {}
func (node *CreateView) nodeStatement() {}

type IfNotExists struct {
	If     Keyword
	Not    Keyword
	Exists Keyword
}

func MakeIfNotExists(
	ifKeyword Keyword,
	notKeyword Keyword,
	existsKeyword Keyword,
) *IfNotExists {
	return &IfNotExists{
		If:     ifKeyword,
		Not:    notKeyword,
		Exists: existsKeyword,
	}
}

func (node *IfNotExists) ToSql(f formatter.Formatter) {
	f.Text(node.If.Text)
	f.Space()
	f.Text(node.Not.Text)
	f.Space()
	f.Text(node.Exists.Text)
}

func (node *IfNotExists) node() {}

type CatalogObjectIdentifier struct {
	SchemaName *Identifier
	ObjectName Identifier
}

func MakeCatalogObjectIdentifier(
	schemaName *Identifier,
	objectName Identifier,
) *CatalogObjectIdentifier {
	return &CatalogObjectIdentifier{
		SchemaName: schemaName,
		ObjectName: objectName,
	}
}

func (node *CatalogObjectIdentifier) node() {}

func (node *CatalogObjectIdentifier) ToSql(f formatter.Formatter) {
	if node.SchemaName != nil {
		node.SchemaName.ToSql(f)
		f.Rune('.')
	}
	node.ObjectName.ToSql(f)
}

func (node *CatalogObjectIdentifier) FullyQualifiedName(defaultSchema string) string {
	schema := defaultSchema
	object := node.ObjectName.Text

	if node.SchemaName != nil {
		schema = node.SchemaName.Text
	}

	return fmt.Sprintf("\"%s\".\"%s\"", schema, object)
}

func (node *CatalogObjectIdentifier) Eq(other *CatalogObjectIdentifier) bool {

	if other == nil {
		return false
	}

	result := true
	if node.SchemaName != nil && other.SchemaName != nil {
		result = result && node.SchemaName.Eq(other.SchemaName)
	} else if node.SchemaName != nil || other.SchemaName != nil {
		return false
	}

	result = result && node.ObjectName.Eq(&other.ObjectName)

	return result
}

type TableDefinition struct {
	LParen tik.Token

	// AsSelect *SelectStatement
	ColumnDefinitions []ColumnDefinition
	TableConstraints  []TableConstraint

	RParent tik.Token
}

func MakeTableDefinition(
	lParen tik.Token,
	columnDefs []ColumnDefinition,
	constraints []TableConstraint,
	rParen tik.Token,
) *TableDefinition {
	return &TableDefinition{
		LParen:            lParen,
		ColumnDefinitions: columnDefs,
		TableConstraints:  constraints,
		RParent:           rParen,
	}
}

func (node *TableDefinition) node() {}

func (node *TableDefinition) ToSql(f formatter.Formatter) {
	f.Anchor(func() {
		for i, col := range node.ColumnDefinitions {
			col.ToSql(f)
			if i < len(node.ColumnDefinitions)-1 {
				f.Rune(',')
				f.Break()
			}
		}
		if len(node.TableConstraints) > 0 {
			f.Rune(',')
			f.Break()
		}
		for i, constraint := range node.TableConstraints {
			constraint.ToSql(f)
			if i < len(node.TableConstraints)-1 {
				f.Rune(',')
				f.Break()
			}
		}
	})
}

type TableOptions struct {
	Strict       *Keyword
	WithoutRowId *WithoutRowId
}

func MakeTableOptions(strict *Keyword, withoutRowId *WithoutRowId) *TableOptions {
	return &TableOptions{
		Strict:       strict,
		WithoutRowId: withoutRowId,
	}
}

func (node *TableOptions) node() {}
func (node *TableOptions) ToSql(f formatter.Formatter) {

}

func (node *TableOptions) IsStrict() bool {
	return node.Strict != nil
}

func (node *TableOptions) IsWithoutRowId() bool {
	return node.WithoutRowId != nil
}

type WithoutRowId struct {
	Without Keyword
	RowId   Keyword
}

func MakeWithoutRowId(without Keyword, rowId Keyword) *WithoutRowId {
	return &WithoutRowId{
		Without: without,
		RowId:   rowId,
	}
}

func (node *WithoutRowId) node() {}

type ColumnDefinition struct {
	ColumnName        Identifier
	TypeName          TypeName
	ColumnConstraints []ColumnConstraint
}

func MakeColumnDefinition(
	name Identifier,
	typ Identifier,
	constraints []ColumnConstraint,
) *ColumnDefinition {
	return &ColumnDefinition{
		ColumnName: name,
		TypeName: TypeName{
			TypeName: typ,
		},
		ColumnConstraints: constraints,
	}
}

func (node *ColumnDefinition) node() {}

func (node *ColumnDefinition) ToSql(f formatter.Formatter) {
	node.ColumnName.ToSql(f)
	f.Space()
	f.Text(node.TypeName.TypeName.Text)

	for _, constraint := range node.ColumnConstraints {
		f.Space()
		constraint.ToSql(f)
	}
}

type TypeName struct {
	TypeName Identifier
}

func (node *TypeName) node() {}

type ConflictClause struct {
	OnKeyword       Keyword
	ConflictKeyword Keyword
	Action          Keyword
}

func MakeConflictClause(
	onKeyword Keyword,
	conflictKeyword Keyword,
	actionKeyword Keyword,
) *ConflictClause {
	return &ConflictClause{
		OnKeyword:       onKeyword,
		ConflictKeyword: conflictKeyword,
		Action:          actionKeyword,
	}
}

func (node *ConflictClause) Eq(other *ConflictClause) bool {
	return node.Action.Eq(&other.Action)
}

type TableConstraint_Check struct {
	Name         *ConstraintName
	CheckKeyword Keyword
	LParen       tik.Token
	Expr         Expr
	RParen       tik.Token
}

func MakeTableConstraintCheck(
	constraintName *ConstraintName,
	checkKeyword Keyword,
	lParen tik.Token,
	expr Expr,
	rParen tik.Token,
) *TableConstraint_Check {
	return &TableConstraint_Check{
		Name:         constraintName,
		CheckKeyword: checkKeyword,
		LParen:       lParen,
		Expr:         expr,
		RParen:       rParen,
	}
}

func (node *TableConstraint_Check) node()                {}
func (node *TableConstraint_Check) nodeTableConstraint() {}

func (node *TableConstraint_Check) Eq(other TableConstraint) bool {

	if otherCheck, ok := other.(*TableConstraint_Check); ok {
		// Assume that named constraints are equivilent
		if node.Name.Eq(otherCheck.Name) {
			return true
		}

		// Otherwise we need to know if the Expression is "equivilent"
		return node.Expr.Eq(otherCheck.Expr)
	}

	return true
}

func (node *TableConstraint_Check) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

type TableConstraint_PrimaryKey struct {
	Name           *ConstraintName
	PrimaryKeyword Keyword
	KeyKeyword     Keyword
	LParen         tik.Token
	IndexedColumns []IndexedColumn
	AutoIncrement  *Keyword
	RParen         tik.Token
	ConflictClause *ConflictClause
}

func MakeTableConstraintPrimaryKey(
	constraintName *ConstraintName,
	primaryKeyword Keyword,
	keyKeyword Keyword,
	lParen tik.Token,
	indexedColumns []IndexedColumn,
	rParen tik.Token,
	conflictClause *ConflictClause,
	// sqlite
	autoincrement *Keyword,

) *TableConstraint_PrimaryKey {
	return &TableConstraint_PrimaryKey{
		Name:           constraintName,
		PrimaryKeyword: primaryKeyword,
		KeyKeyword:     keyKeyword,
		LParen:         lParen,
		IndexedColumns: indexedColumns,
		AutoIncrement:  autoincrement,
		RParen:         rParen,
		ConflictClause: conflictClause,
	}
}

func (node *TableConstraint_PrimaryKey) node()                {}
func (node *TableConstraint_PrimaryKey) nodeTableConstraint() {}
func (node *TableConstraint_PrimaryKey) Eq(other TableConstraint) bool {
	if other, ok := other.(*TableConstraint_PrimaryKey); ok {

		if len(node.IndexedColumns) != len(other.IndexedColumns) {
			return false
		}

		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		for i, column := range node.IndexedColumns {
			otherColumn := &other.IndexedColumns[i]
			result = result && column.Eq(otherColumn)
		}

		if node.ConflictClause != nil && other.ConflictClause != nil {
			result = result && node.ConflictClause.Eq(other.ConflictClause)
		} else if node.ConflictClause != nil || other.ConflictClause != nil {
			return false
		}

		return true
	}
	return false
}

func (node *TableConstraint_PrimaryKey) ToSql(f formatter.Formatter) {
	f.Group(func() {
		if node.Name != nil {
			node.Name.ToSql(f)
			f.Space()
		}

		f.Text("PRIMARY")
		f.Space()
		f.Text("KEY")
		f.Space()
		f.Rune('(')

		if node.AutoIncrement != nil {
			col := node.IndexedColumns[0]
			col.ToSql(f)
			f.Space()
			f.Text("AUTOINCREMENT")
		} else {
			for i, col := range node.IndexedColumns {

				col.ToSql(f)

				if i < len(node.IndexedColumns)-1 {
					f.Rune(',')
					f.Space()
				}
			}
		}

		f.Rune(')')
	})
}

type TableConstraint_ForeignKey struct {
	Name           *ConstraintName
	ForeignKeyword Keyword
	KeyKeyword     Keyword
	LParen         tik.Token
	Columns        []Identifier
	RParen         tik.Token
	FkClause       ForeignKeyClause
}

func MakeTableConstraintForeignKey(
	constraintName *ConstraintName,
	foreignKeyword Keyword,
	keyKeyword Keyword,
	lParen tik.Token,
	columns []Identifier,
	rParen tik.Token,
	fkClause *ForeignKeyClause,
) *TableConstraint_ForeignKey {
	return &TableConstraint_ForeignKey{
		Name:           constraintName,
		ForeignKeyword: foreignKeyword,
		KeyKeyword:     keyKeyword,
		LParen:         lParen,
		Columns:        columns,
		RParen:         rParen,
		FkClause:       *fkClause,
	}
}

func (node *TableConstraint_ForeignKey) node()                {}
func (node *TableConstraint_ForeignKey) nodeTableConstraint() {}
func (node *TableConstraint_ForeignKey) Eq(other TableConstraint) bool {
	if other, ok := other.(*TableConstraint_ForeignKey); ok {

		if node.Name != nil && other.Name != nil {
			return node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		thisPairs := node.pairColumns()
		otherPairs := other.pairColumns()

		if len(thisPairs) != len(otherPairs) {
			return false
		}

		result := true

		for i := range len(thisPairs) {
			thisPair := thisPairs[i]
			otherPair := otherPairs[i]

			result = result && thisPair.Eq(otherPair)
		}

		result = result && node.FkClause.Eq(&other.FkClause)
		return result
	}
	return false
}

func (node *TableConstraint_ForeignKey) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

type IdentifierPair struct {
	A, B Identifier
}

func (this IdentifierPair) Eq(other IdentifierPair) bool {
	return this.A.Eq(&other.A) && this.B.Eq(&other.B)
}

func (node *TableConstraint_ForeignKey) pairColumns() []IdentifierPair {

	pairs := make([]IdentifierPair, 0, len(node.Columns))

	for i, localCol := range node.Columns {
		foreignCol := node.FkClause.ForeignColumns[i]
		pairs = append(pairs, IdentifierPair{
			A: localCol,
			B: foreignCol,
		})
	}

	return pairs
}

type ForeignKeyClause struct {
	ReferencesKeyword Keyword
	ForeignTable      CatalogObjectIdentifier
	LParen            tik.Token
	ForeignColumns    []Identifier
	RParen            tik.Token
	Actions           []ForeignKeyAction
	MatchName         *Identifier
	Deferrable        *ForeignKeyDeferrable
}

func MakeForeignKeyClause(
	referencesKeyword Keyword,
	foreignTable CatalogObjectIdentifier,
	lParen tik.Token,
	foreignColumns []Identifier,
	rParen tik.Token,
	actions []ForeignKeyAction,
	matchName *Identifier,
	deferrable *ForeignKeyDeferrable,
) *ForeignKeyClause {
	return &ForeignKeyClause{
		ReferencesKeyword: referencesKeyword,
		ForeignTable:      foreignTable,
		LParen:            lParen,
		ForeignColumns:    foreignColumns,
		RParen:            rParen,
		Actions:           actions,
		MatchName:         matchName,
		Deferrable:        deferrable,
	}
}

func (node *ForeignKeyClause) node() {}
func (node *ForeignKeyClause) Eq(other *ForeignKeyClause) bool {
	if len(node.ForeignColumns) != len(other.ForeignColumns) {
		return false
	}
	if len(node.Actions) != len(other.Actions) {
		return false
	}

	result := true
	result = result && node.ForeignTable.Eq(&other.ForeignTable)

	cmp := func(a, b Identifier) int {
		if a.Text < b.Text {
			return -1
		}
		if a.Text > b.Text {
			return 1
		}
		return 0
	}

	aCols := slices.SortedFunc(slices.Values(node.ForeignColumns), cmp)
	bCols := slices.SortedFunc(slices.Values(other.ForeignColumns), cmp)

	for i, foreignCol := range aCols {
		result = result && foreignCol.Eq(&bCols[i])
	}

	return result
}

type ForeignKeyDeferrable struct {
	NotKeyword        *Keyword
	DeferrableKeyword Keyword
	InitiallyKeyword  *Keyword
	Deferrable        *Keyword
}

func MakeForeignKeyDeferrable(notKeyword *Keyword, deferrableKeyword Keyword, initiallyKeyword *Keyword, value *Keyword) *ForeignKeyDeferrable {
	return &ForeignKeyDeferrable{
		NotKeyword:        notKeyword,
		DeferrableKeyword: deferrableKeyword,
		InitiallyKeyword:  initiallyKeyword,
		Deferrable:        value,
	}
}

func (node *ForeignKeyDeferrable) node() {}

type ForeignKeyAction interface {
	nodeForeignKeyAction()
}

type ForeignKeyDeleteAction struct {
	OnKeyword     Keyword
	DeleteKeyword Keyword
	Action        ForeignKeyActionDo
}

func MakeForeignKeyDeleteAction(
	onKeyword Keyword,
	deleteKeyword Keyword,
	do ForeignKeyActionDo,
) *ForeignKeyDeleteAction {
	return &ForeignKeyDeleteAction{
		OnKeyword:     onKeyword,
		DeleteKeyword: deleteKeyword,
		Action:        do,
	}
}

type ForeignKeyUpdateAction struct {
	OnKeyword     Keyword
	UpdateKeyword Keyword
	Action        ForeignKeyActionDo
}

func (node *ForeignKeyDeleteAction) nodeForeignKeyAction() {}

func MakeForeignKeyUpdateAction(
	onKeyword Keyword,
	updateKeyword Keyword,
	do ForeignKeyActionDo,
) *ForeignKeyUpdateAction {
	return &ForeignKeyUpdateAction{
		OnKeyword:     onKeyword,
		UpdateKeyword: updateKeyword,
		Action:        do,
	}
}

func (node *ForeignKeyUpdateAction) nodeForeignKeyAction() {}

type ForeignKeyActionDo interface {
	AstNode
	nodeForeignKeyActionDo()
	Eq(other ForeignKeyActionDo) bool
}

type NoAction struct {
	NoKeyword     Keyword
	ActionKeyword Keyword
}

func MakeForeignKeyActionNoAction(noKeyword Keyword, actionKeyword Keyword) *NoAction {
	return &NoAction{
		NoKeyword:     noKeyword,
		ActionKeyword: actionKeyword,
	}
}

func (node *NoAction) node()                   {}
func (node *NoAction) nodeForeignKeyActionDo() {}
func (node *NoAction) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*NoAction)
	return ok
}

type Restrict Keyword

func MakeForeignKeyActionRestrict(restrictKeyword Keyword) *Restrict {
	val := Restrict(restrictKeyword)
	return &val
}

func (node *Restrict) node()                   {}
func (node *Restrict) nodeForeignKeyActionDo() {}
func (node *Restrict) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*Restrict)
	return ok
}

type SetNull struct {
	SetKeyword  Keyword
	NullKeyword Keyword
}

func MakeForeignKeyActionSetNull(setKeyword Keyword, nullKeyword Keyword) *SetNull {
	return &SetNull{
		SetKeyword:  setKeyword,
		NullKeyword: nullKeyword,
	}
}

func (node *SetNull) node()                   {}
func (node *SetNull) nodeForeignKeyActionDo() {}
func (node *SetNull) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*SetNull)
	return ok
}

type SetDefault struct {
	SetKeyword     Keyword
	DefaultKeyword Keyword
}

func MakeForeignKeyActionSetDefault(setKeyword Keyword, defaultKeyword Keyword) *SetDefault {
	return &SetDefault{
		SetKeyword:     setKeyword,
		DefaultKeyword: defaultKeyword,
	}
}

func (node *SetDefault) node()                   {}
func (node *SetDefault) nodeForeignKeyActionDo() {}
func (node *SetDefault) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*SetDefault)
	return ok
}

type Cascade Keyword

func MakeForeignKeyActionCascade(cascadeKeyword Keyword) *Cascade {
	val := Cascade(cascadeKeyword)
	return &val
}

func (node *Cascade) node()                   {}
func (node *Cascade) nodeForeignKeyActionDo() {}
func (node *Cascade) Eq(other ForeignKeyActionDo) bool {
	_, ok := other.(*Cascade)
	return ok
}

type ConstraintName struct {
	ConstraintKeyword Keyword
	Name              Identifier
}

func (node *ConstraintName) ToSql(f formatter.Formatter) {
	node.ConstraintKeyword.ToSql(f)
	f.Space()
	node.Name.ToSql(f)
}

func (node *ConstraintName) Eq(other *ConstraintName) bool {
	return node.Name.Eq(other.Name.AsExpr())
}

type ColumnConstraint_PrimaryKey struct {
	Name           *ConstraintName
	PrimaryKeyword Keyword
	KeyKeyword     Keyword
	Order          *Keyword
	ConflictClause *ConflictClause

	// sqlite autoincrement
	AutoIncrement *Keyword
}

func MakeColumnConstraintPrimaryKey(
	constraintName *ConstraintName,
	primaryKeyword Keyword,
	keyKeyword Keyword,
	order *Keyword,
	conflictClause *ConflictClause,
	autoIncrement *Keyword,
) *ColumnConstraint_PrimaryKey {
	return &ColumnConstraint_PrimaryKey{
		Name:           constraintName,
		PrimaryKeyword: primaryKeyword,
		KeyKeyword:     keyKeyword,
		Order:          order,
		ConflictClause: conflictClause,
		AutoIncrement:  autoIncrement,
	}
}

func (node *ColumnConstraint_PrimaryKey) node()                 {}
func (node *ColumnConstraint_PrimaryKey) nodeColumnConstraint() {}

func (node *ColumnConstraint_PrimaryKey) IsAutoIncrement() bool {
	return node.AutoIncrement != nil
}

func (node *ColumnConstraint_PrimaryKey) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_PrimaryKey); ok {
		result := true

		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		return result
	}
	return false
}

func (node *ColumnConstraint_PrimaryKey) ToSql(f formatter.Formatter) {
	if node.Name != nil {
		f.Text("")
	}
}

type ColumnConstraint_Unique struct {
	Name *ConstraintName
}

func (node *ColumnConstraint_Unique) node()                 {}
func (node *ColumnConstraint_Unique) nodeColumnConstraint() {}
func (node *ColumnConstraint_Unique) Eq(other ColumnConstraint) bool {
	_, ok := other.(*ColumnConstraint_Unique)
	return ok
}

func (node *ColumnConstraint_Unique) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

type ColumnConstraint_Collate struct {
	Name    *ConstraintName
	Collate Identifier
}

func (node *ColumnConstraint_Collate) node()                 {}
func (node *ColumnConstraint_Collate) nodeColumnConstraint() {}
func (node *ColumnConstraint_Collate) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_Collate); ok {
		result := true

		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		result = result && node.Collate.Eq(&other.Collate)
		return result
	}
	return false
}

func (node *ColumnConstraint_Collate) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

type ColumnConstraint_NotNull struct {
	Name *ConstraintName
}

func (node *ColumnConstraint_NotNull) node()                 {}
func (node *ColumnConstraint_NotNull) nodeColumnConstraint() {}
func (node *ColumnConstraint_NotNull) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_NotNull); ok {
		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}
		return result
	}
	return false
}

func (node *ColumnConstraint_NotNull) ToSql(f formatter.Formatter) {
	if node.Name != nil {
		f.Text("CONSTRAINT")
		f.Space()
		node.Name.ToSql(f)
		f.Space()
	}

	f.Text("NOT")
	f.Space()
	f.Text("NULL")
}

type ColumnConstraint_Default struct {
	Name    *ConstraintName
	Default Expr
}

func (node *ColumnConstraint_Default) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *ColumnConstraint_Default) node()                 {}
func (node *ColumnConstraint_Default) nodeColumnConstraint() {}
func (node *ColumnConstraint_Default) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_Default); ok {
		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}
		return result
	}
	return false
}

type ColumnConstraint_Generated struct {
	Name    *ConstraintName
	As      Expr
	Storage AstNode
}

func (node *ColumnConstraint_Generated) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *ColumnConstraint_Generated) node()                 {}
func (node *ColumnConstraint_Generated) nodeColumnConstraint() {}
func (node *ColumnConstraint_Generated) Eq(other ColumnConstraint) bool {
	if other, ok := other.(*ColumnConstraint_Generated); ok {
		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}
		// @NOTE(woody): do we care about Storage?
		return result
	}
	return false
}

type ColumnConstraint_Check struct {
	Name  *ConstraintName
	Check Expr
}

func (node *ColumnConstraint_Check) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *ColumnConstraint_Check) node()                 {}
func (node *ColumnConstraint_Check) nodeColumnConstraint() {}

func (node *ColumnConstraint_Check) Eq(other ColumnConstraint) bool {

	if other, ok := other.(*ColumnConstraint_Check); ok {
		// Assume that named constraints are equivilent
		result := true
		if node.Name != nil && other.Name != nil {
			result = result && node.Name.Eq(other.Name)
		} else if node.Name != nil || other.Name != nil {
			return false
		}

		// Otherwise we need to know if the Expression is "equivilent"
		return node.Check.Eq(other.Check)
	}

	return true
}

type ExprList []Expr

func (node ExprList) node()           {}
func (node ExprList) nodeExpression() {}
func (node ExprList) Eq(other Expr) bool {
	if otherExprList, ok := other.(ExprList); ok {

		if len(node) != len(otherExprList) {
			return false
		}

		result := true
		for i := range len(node) {
			a := node[i]
			b := otherExprList[i]
			result = result && a.Eq(b)
		}

		return result
	}
	return false
}

type LiteralNull struct {
	Token tik.Token
}

func (node *LiteralNull) node()           {}
func (node *LiteralNull) nodeExpression() {}
func (node *LiteralNull) nodeLiteral()    {}
func (node *LiteralNull) Eq(other Expr) bool {
	if _, ok := other.(*LiteralNull); ok {
		return true
	}
	return false
}

var ErrTokenUnconvertableToBoolean = errors.New("token is not convertable to boolean")

func TokenToLiteralBoolean(token tik.Token) (LiteralBoolean, error) {
	switch token.Kind {
	case tik.TokenKind_DecimalNumericLiteral:
		fVal, err := strconv.ParseFloat(token.Text, 64)
		if err != nil {

			return LiteralBoolean{
				Token: token,
				Value: false,
			}, fmt.Errorf("%w: %w :token is %s", ErrTokenUnconvertableToBoolean, err, token.Text)
		}
		if fVal > 0 {
			return LiteralBoolean{
				Token: token,
				Value: true,
			}, nil
		} else {
			return LiteralBoolean{
				Token: token,
				Value: false,
			}, nil
		}
	case tik.TokenKind_Keyword_TRUE:
		return LiteralBoolean{
			Token: token,
			Value: true,
		}, nil
	case tik.TokenKind_Keyword_FALSE:
		return LiteralBoolean{
			Token: token,
			Value: false,
		}, nil
	case tik.TokenKind_Identifier:
		if strings.ToLower(token.Text) == "true" {
			return LiteralBoolean{
				Token: token,
				Value: true,
			}, nil
		} else if strings.ToLower(token.Text) == "false" {
			return LiteralBoolean{
				Token: token,
				Value: false,
			}, nil
		} else {
			return LiteralBoolean{}, fmt.Errorf("%w: unknown identifier: token is %s", ErrTokenUnconvertableToBoolean, token.Text)
		}
	default:
		return LiteralBoolean{}, fmt.Errorf("%w: unexpected token: token is %s", ErrTokenUnconvertableToBoolean, token.Text)
	}
}

type LiteralBoolean struct {
	Token tik.Token
	Value bool
}

func (node *LiteralBoolean) ToSql(f formatter.Formatter) {
	f.Text(node.Token.Text)
}

func (node *LiteralBoolean) node()            {}
func (node *LiteralBoolean) nodeExpression()  {}
func (node *LiteralBoolean) nodeLiteral()     {}
func (node *LiteralBoolean) nodePragmaValue() {}
func (node *LiteralBoolean) Eq(other Expr) bool {
	if otherBool, ok := other.(*LiteralBoolean); ok {
		return node.Value == otherBool.Value
	}
	return false
}

func TokenToLiteralInteger(token tik.Token) (LiteralInteger, error) {
	panic("not implemented")
}

type LiteralInteger struct {
	Token tik.Token
	Value int64
}

func (node *LiteralInteger) node()            {}
func (node *LiteralInteger) nodeExpression()  {}
func (node *LiteralInteger) nodeLiteral()     {}
func (node *LiteralInteger) nodePragmaValue() {}
func (node *LiteralInteger) Eq(other Expr) bool {
	if otherNumber, ok := other.(*LiteralInteger); ok {
		return otherNumber.Token.Text == node.Token.Text
	}
	return false
}

type LiteralFloat struct {
	Token tik.Token
	Value float64
}

func (node *LiteralFloat) node()            {}
func (node *LiteralFloat) nodeExpression()  {}
func (node *LiteralFloat) nodeLiteral()     {}
func (node *LiteralFloat) nodePragmaValue() {}
func (node *LiteralFloat) Eq(other Expr) bool {
	if otherFloat, ok := other.(*LiteralFloat); ok {
		// we check the text value here as to not compare floats.
		return otherFloat.Token.Text == node.Token.Text
	}
	return false
}

type LiteralString struct {
	Token tik.Token
	Value string
}

func (node *LiteralString) node()            {}
func (node *LiteralString) nodeExpression()  {}
func (node *LiteralString) nodeLiteral()     {}
func (node *LiteralString) nodePragmaValue() {}
func (node *LiteralString) Eq(other Expr) bool {
	if otherString, ok := other.(*LiteralString); ok {
		return node.Value == otherString.Value
	}
	return false
}

type UnaryOperator struct {
	Operator tik.Token
	Rhs      Expr
}

func (node *UnaryOperator) node()           {}
func (node *UnaryOperator) nodeExpression() {}

type FunctionCall struct {
	Name Identifier
	Args ExprList
}

func (node *FunctionCall) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *FunctionCall) node()           {}
func (node *FunctionCall) nodeExpression() {}
func (node *FunctionCall) Eq(other Expr) bool {
	if otherFn, ok := other.(*FunctionCall); ok {
		result := true
		result = result && node.Name.Eq(&otherFn.Name)
		result = result && node.Args.Eq(otherFn.Args)
		return result
	}
	return false
}

type ColumnName struct {
	Schema *Identifier
	Table  *Identifier
	Column Identifier
}

func (node *ColumnName) ToSql(f formatter.Formatter) {
	panic("not implemented")
}

func (node *ColumnName) node()           {}
func (node *ColumnName) nodeExpression() {}
func (node *ColumnName) Eq(other Expr) bool {
	if other, ok := other.(*ColumnName); ok {
		result := true

		if node.Schema != nil && other.Schema != nil {
			result = result && node.Schema.Eq(other.Schema)
		} else if node.Schema != nil || other.Schema != nil {
			return false
		}

		if node.Table != nil && other.Table != nil {
			result = result && node.Table.Eq(other.Table)
		} else if node.Table != nil || other.Table != nil {
			return false
		}

		result = result && node.Column.Eq(&other.Column)

		return result
	}
	return false
}

type BindingPower struct {
	L int
	R int
}

type BinaryOp struct {
	Operator tik.Token
	Lhs      Expr
	Rhs      Expr
}

func MakeBinaryOpExpr(
	lhs Expr,
	op tik.Token,
	rhs Expr,
) *BinaryOp {
	return &BinaryOp{
		Operator: op,
		Lhs:      lhs,
		Rhs:      rhs,
	}
}

func (node *BinaryOp) node()           {}
func (node *BinaryOp) nodeExpression() {}
func (node *BinaryOp) Eq(other Expr) bool {
	if other, ok := other.(*BinaryOp); ok {
		if node.Operator.Text != other.Operator.Text {
			return false
		}

		result := true
		result = result && node.Lhs.Eq(node.Lhs)
		result = result && node.Rhs.Eq(node.Rhs)
		return result
	}
	return false
}

type CaseExpression struct {
	Operand Expr
	Cases   []WhenThen
	Else    Expr
}

func (node *CaseExpression) node()           {}
func (node *CaseExpression) nodeExpression() {}
func (node *CaseExpression) Eq(other Expr) bool {

	if other, ok := other.(*CaseExpression); ok {
		if len(node.Cases) != len(other.Cases) {
			return false
		}

		result := true
		result = result && node.Operand.Eq(other.Operand)

		for i := range len(node.Cases) {
			aCase := node.Cases[i]
			bCase := other.Cases[i]

			result = result && aCase.When.Eq(bCase.When)
			result = result && aCase.Then.Eq(bCase.Then)
		}

		result = result && node.Else.Eq(other.Else)
		return result
	}
	return false

}

type WhenThen struct {
	When Expr
	Then Expr
}

func (node *WhenThen) node() {}

type Collation struct {
	CollateKeyword Keyword
	Name           Identifier
}

func MakeCollation(
	collateKeyword Keyword,
	name Identifier,
) *Collation {
	return &Collation{
		CollateKeyword: collateKeyword,
		Name:           name,
	}
}
