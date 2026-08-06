package op

import (
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/frontend/ast"
)

type Op interface {
	isOp()
	ast.Equalable
}

func (*MigrateData) isOp()          {}
func (*AddTable) isOp()             {}
func (*DropTable) isOp()            {}
func (*RenameTable) isOp()          {}
func (*AddTableConstraint) isOp()   {}
func (*DropTableConstraint) isOp()  {}
func (*AddColumn) isOp()            {}
func (*DropColumn) isOp()           {}
func (*RenameColumn) isOp()         {}
func (*ChangeColumnType) isOp()     {}
func (*AddColumnConstraint) isOp()  {}
func (*DropColumnConstraint) isOp() {}

type AddTable struct {
	Target TargetSchema
	Table  *schema.Table
}

type DropTable struct {
	Target TargetTable
}

type RenameTable struct {
	Target  TargetTable
	NewName *ast.CatalogObjectIdentifier
}

type AddTableConstraint struct {
	Target     TargetTable
	Name       ast.Identifier
	Constraint ast.TableConstraint
}

type DropTableConstraint struct {
	Target TargetTableConstraint
}

type AddColumn struct {
	Target TargetTable
	Column schema.ColumnLike
}

type DropColumn struct {
	Target TargetColumn
}

type RenameColumn struct {
	Target  TargetColumn
	NewName *ast.Identifier
}

type ChangeColumnType struct {
	Target  TargetColumn
	NewType *ast.TypeName
}

type AddColumnConstraint struct {
	Target     TargetColumn
	Constraint ast.ColumnConstraint
}

type DropColumnConstraint struct {
	Target TargetColumnConstraint
}

type MigrateData struct {
	SrcTarget TargetTable
	DstTarget TargetTable
	Ops       []Op
}

func MakeDropConstraintFromPrimaryKey(schem *schema.Schema, table *schema.Table, pk *schema.PrimaryKey) Op {
	switch node := pk.Node.(type){
	case *ast.ColumnConstraint_PrimaryKey:
		return &DropColumnConstraint {
			Target: TargetColumnConstraint{
				Schema: schem,
				Table: table,
				Column: pk.ResolvedColumns[0],
				Constraint: node,
			},
		}
	case *ast.TableConstraint_PrimaryKey:
		return &DropTableConstraint{
			Target: TargetTableConstraint{
				Schema: schem,
				Table: table,
				Constraint: node,
			},
		}
	default:
		panic("invalid node type")
	}			
}

func MakeAddConstraintFromPrimaryKey(schem *schema.Schema, table *schema.Table, pk *schema.PrimaryKey) Op {
	switch node := pk.Node.(type){
	case *ast.ColumnConstraint_PrimaryKey:
		return &AddColumnConstraint {
			Target: TargetColumn{
				Schema: schem,
				Table: table,
				Column: pk.ResolvedColumns[0],
			},
			Constraint: node,
		}
	case *ast.TableConstraint_PrimaryKey:
		return &AddTableConstraint{
			Target: TargetTable{
				Schema: schem,
				Table: table,
			},
			Constraint: node,
		}
	default:
		panic("invalid node type")
	}			
}
