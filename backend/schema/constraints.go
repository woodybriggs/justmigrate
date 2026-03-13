package schema

import "woodybriggs/justmigrate/core/ast"

type PrimaryKeyCommon struct {
}

type PrimaryKey struct {
	Node interface {
		Accept(ast.Visitor)
	}
	Order          Order
	ConflictClause ConflictAction
}

type ForeignKey struct {
}
