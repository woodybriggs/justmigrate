package op

import (
	"woodybriggs/justmigrate/backend/schema"
	"woodybriggs/justmigrate/frontend/ast"
)

type TargetSchema struct {
	Schema *schema.Schema
}

type TargetTable struct {
	Schema *schema.Schema
	Table  *schema.Table
}

type TargetTableConstraint struct {
	Schema     *schema.Schema
	Table      *schema.Table
	Constraint ast.TableConstraint
}

type TargetColumn struct {
	Schema *schema.Schema
	Table  *schema.Table
	Column schema.ColumnLike
}

type TargetColumnConstraint struct {
	Schema     *schema.Schema
	Table      *schema.Table
	Column     schema.ColumnLike
	Constraint ast.ColumnConstraint
}

type TargetIndex struct {
	Schema *schema.Schema
	Table  *schema.Table
	Index  *schema.Index
}
