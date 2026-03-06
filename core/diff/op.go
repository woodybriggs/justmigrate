package diff

import "woodybriggs/justmigrate/core/ast"

type Op interface {
	op()
}

func (*NewTableOp) op()      {}
func (*DelTableOp) op()      {}
func (*RenameTableOp) op()   {}
func (*NewColOp) op()        {}
func (*DelColOp) op()        {}
func (*RenameColOp) op()     {}
func (*ChangeColTypeOp) op() {}

type NewTableOp struct {
	*ast.CreateTable
}

type DelTableOp struct {
	*ast.CatalogObjectIdentifier
}

type RenameTableOp struct {
	From *ast.CatalogObjectIdentifier
	To   *ast.CatalogObjectIdentifier
}

type DelColOp struct {
	Table *ast.CatalogObjectIdentifier
	Col   *ast.Identifier
}

type NewColOp struct {
	Table *ast.CatalogObjectIdentifier
	Col   *ast.ColumnDefinition
}

type RenameColOp struct {
	Table   *ast.CatalogObjectIdentifier
	FromCol *ast.Identifier
	ToCol   *ast.Identifier
}

type ChangeColTypeOp struct {
	Table    *ast.CatalogObjectIdentifier
	Col      *ast.Identifier
	TypeName *ast.TypeName
}
