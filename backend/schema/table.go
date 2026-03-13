package schema

import "woodybriggs/justmigrate/core/ast"

type Table struct {
	Node *ast.CreateTable

	Name    string
	Columns map[string]*Column
}

func TableFromAst(createTable *ast.CreateTable) *Table {

	return &Table{
		Node: createTable,
		Name: createTable.TableIdentifier.String(),
	}
}
