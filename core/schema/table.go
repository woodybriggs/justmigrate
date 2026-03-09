package schema

import "woodybriggs/justmigrate/core/ast"

type Table struct {
	Identifier *ast.CatalogObjectIdentifier
}

func TableFromAst(createTable *ast.CreateTable) (*Table, error) {
	return &Table{}, nil
}
