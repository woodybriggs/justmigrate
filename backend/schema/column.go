package schema

import (
	"errors"
	"woodybriggs/justmigrate/core/ast"
)

type Type struct {
	TypeName *ast.TypeName

	Name string
	Args []ast.NumericLiteral
}

func TypeFromAst(typeName *ast.TypeName) *Type {
	if typeName == nil {
		return nil
	}
	return &Type{
		TypeName: typeName,
		Name:     typeName.Name.Text,
		Args:     []ast.NumericLiteral{typeName.Arg0, typeName.Arg1},
	}
}

func PrimaryKeyFromColumnConstraint(constraint *ast.ColumnConstraint_PrimaryKey) *PrimaryKey {
	panic(errors.New("not implemented"))
}

func ForeignKeyFromColumnConstraint(constraint *ast.ColumnConstraint_ForeignKey) *ForeignKey {
	panic(errors.New("not implemented"))
}

type ColumnConstraints struct {
	Collate     string
	NotNull     bool
	Unique      bool
	Checks      []ast.Expr
	FK          any
	PK          any
	Default     ast.Expr
	GeneratedAs ast.Expr
}

func ColumnConstraintsFromAst(constraints []ast.ColumnConstraint) *ColumnConstraints {

	var notNull bool
	var unique bool
	var collate string
	var checks []ast.Expr
	var defaultVal ast.Expr
	var generatedAsExpr ast.Expr
	var fk any
	var pk *PrimaryKey

	for _, constraint := range constraints {
		switch typ := constraint.(type) {
		case *ast.ColumnConstraint_Check:
			checks = append(checks, typ.CheckExpr)
		case *ast.ColumnConstraint_Collate:
			collate = typ.CollationName.Text
		case *ast.ColumnConstraint_Generated:
			generatedAsExpr = typ.AsExpr
		case *ast.ColumnConstraint_NotNull:
			notNull = true
		case *ast.ColumnConstraint_Default:
			defaultVal = typ.Default
		case *ast.ColumnConstraint_Unique:
			unique = true
		case *ast.ColumnConstraint_PrimaryKey:
			pk = PrimaryKeyFromColumnConstraint(typ)
		case *ast.ColumnConstraint_ForeignKey:
			fk = ForeignKeyFromColumnConstraint(typ)
		}
	}

	return &ColumnConstraints{
		Collate:     collate,
		NotNull:     notNull,
		Unique:      unique,
		Checks:      checks,
		PK:          pk,
		FK:          fk,
		GeneratedAs: generatedAsExpr,
		Default:     defaultVal,
	}
}

type Columnish interface {
	column()
}

func (genCol *GeneratedColumn) column() {}
func (col *Column) column()             {}

type GeneratedColumn struct {
	ColumnDefinition *ast.ColumnDefinition

	Name    string
	Type    *Type
	Storage Storage
	ColumnConstraints
}

type Column struct {
	ColumnDefinition *ast.ColumnDefinition

	Name string
	Type *Type
	PK   any
	ColumnConstraints
}

func ColumnFromAst(colDef *ast.ColumnDefinition) Columnish {
	return &Column{
		ColumnDefinition:  colDef,
		Name:              colDef.ColumnName.Text,
		Type:              TypeFromAst(colDef.TypeName),
		ColumnConstraints: *ColumnConstraintsFromAst(colDef.ColumnConstraints),
	}
}
