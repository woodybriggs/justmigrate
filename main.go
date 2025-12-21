package main

import (
	"fmt"
	"os"
)

type Diagnostic struct {
	OgError *Error
}

func NewDiagnostic(ogError *Error) Diagnostic {
	return Diagnostic{
		OgError: ogError,
	}
}

type SyntaxValidator struct {
	Diagnostics []Diagnostic
}

func (sv *SyntaxValidator) ValidateSyntax(nodes []AstNode) []Diagnostic {
	sv.Diagnostics = []Diagnostic{}
	for i := range nodes {
		sv.validateNode(nodes[i])
	}
	return sv.Diagnostics
}

func (sv *SyntaxValidator) validateNode(node AstNode) {
	switch n := node.(type) {
	case *Error:
		sv.Diagnostics = append(sv.Diagnostics, NewDiagnostic(n))
	case *Pragma:
		sv.validateNode(n.Name)
		sv.validateNode(n.Value)
	case *CreateTable:
		sv.validateNode(n.TableIdentifier)
		sv.validateNode(n.TableDefinition)
		sv.validateNode(n.TableOptions)
	case *CreateVirtualTable:
		sv.validateNode(n.TableIdentifier)
		sv.validateNode(n.ModuleName)
	case *CreateIndex:
		sv.validateNode(n.IndexIdentifier)
		sv.validateNode(n.OnTable)
		for i := range n.IndexedColumns {
			sv.validateNode(n.IndexedColumns[i])
		}
		if n.WhereExpr != nil {
			sv.validateNode(n.WhereExpr)
		}
	case *CreateView:
		sv.validateNode(n.ViewIdentifier)
		for i := range n.Columns {
			sv.validateNode(n.Columns[i])
		}
		sv.validateNode(n.AsSelect)
	case *IndexedColumn:
		sv.validateNode(n.Subject)
		if n.Order != nil {
			sv.validateNode(n.Order)
		}
		if n.CollationName != nil {
			sv.validateNode(n.CollationName)
		}
	case *CatalogObjectIdentifier:
		if n.SchemaName != nil {
			sv.validateNode(n.SchemaName)
		}
		sv.validateNode(n.ObjectName)
	case *TableDefinition:
		for i := range n.ColumnDefinitions {
			sv.validateNode(n.ColumnDefinitions[i])
		}
		for i := range n.TableConstraints {
			sv.validateNode(n.ColumnDefinitions[i])
		}
	case *TableOptions:
		break
	case *ColumnDefinition:
		sv.validateNode(n.ColumnName)
		sv.validateNode(n.TypeName)
		for i := range n.ColumnConstraints {
			sv.validateNode(n.ColumnConstraints[i])
		}
	case *TypeName:
		sv.validateNode(n.TypeName)
	case *Token:
		break
	case *LiteralBoolean:
		break
	case *LiteralString:
		break
	case *LiteralNumber:
		break
	case *BeginTransaction:
		break
	case *CommitTransaction:
		break
	case *ColumnConstraint_PrimaryKey:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
		if n.Order != nil {
			sv.validateNode(n.Order)
		}
		if n.ConflictClause != nil {
			sv.validateNode(n.ConflictClause)
		}
	case *ColumnConstraint_NotNull:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
	case *ColumnConstraint_Unique:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
	case *ColumnConstraint_Collate:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
		sv.validateNode(n.Collate)
	case *ColumnConstraint_Default:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
		sv.validateNode(n.Default)
	case *ColumnConstraint_Generated:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
		sv.validateNode(n.As)
	case *Constraint_Check:
		if n.Name != nil {
			sv.validateNode(n.Name)
		}
		sv.validateNode(n.Check)
	case *SubOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *MulOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *DivOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *EquivOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *InOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *GteOp:
		sv.validateNode(n.Lhs)
		sv.validateNode(n.Rhs)
	case *FunctionCall:
		sv.validateNode(n.Args)
	case ExprList:
		for _, expr := range n {
			sv.validateNode(expr)
		}
	case *Select:
		break
	case *CreateTrigger:
		break
	case *CaseExpression:
		if n.Operand != nil {
			sv.validateNode(n.Operand)
		}
		for _, case_ := range n.Cases {
			sv.validateNode(case_.When)
			sv.validateNode(case_.Then)
		}
		if n.Else != nil {
			sv.validateNode(n.Else)
		}
	default:
		fmt.Printf("not implemented: %T %+v \n", n, n)
		os.Exit(1)
	}
}
func main() {

	filename := "./resources/test.sql"

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	tokenizer, err := NewTokenizerFromFile(file)
	if err != nil {
		panic(err)
	}

	parser := NewParser(tokenizer)
	stmts := parser.Statements()
	validator := SyntaxValidator{}

	diags := validator.ValidateSyntax(stmts)

	for _, stmt := range stmts {
		fmt.Println(stmt)
	}

	renderer := Renderer{}
	for _, diag := range diags {
		report := NewErrorReport().
			WithCode(0).
			WithMessage(diag.OgError.Error()).
			WithLabels([]Label{
				{Source: diag.OgError.SourceCode, Range: diag.OgError.OffendingToken.SourceRange, Note: ""},
			})
		fmt.Println(renderer.Render(report))
	}
}
