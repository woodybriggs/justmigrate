package generator

import (
	"fmt"
	"woodybriggs/justmigrate/backend/op"
	"woodybriggs/justmigrate/backend/schema"
	"woodybriggs/justmigrate/errext"
)

/*
Plan takes a first pass of "dumb" operations (create table, drop table, etc.)
and transforms them into a detailed, ordered, and valid execution plan for SQLite.

This process is particularly complex for SQLite due to its limited `ALTER TABLE`
support, which often necessitates a full table recreation for many common modifications.

The planner's responsibilities include:

 1. Target Grouping & Squashing: Group incoming operations by their target table.
    Evaluate the batch of operations for each table to determine if any single
    operation requires a full table recreation. If so, squash all operations
    for that table (even natively supported ones) into a single logical mutation
    to prevent redundant disk writes and intermediate invalid states.

 2. State Diffing & Lowering: Convert unsupported high-level operations into
    dialect-supported SQL. For SQLite, this frequently triggers the "12-step"
    table recreation strategy. The planner calculates the exact "Before" and
    "After" table states to generate the sequence:
    - Creating a temporary new table with the final desired schema.
    - Generating an `INSERT INTO ... SELECT ...` statement, safely mapping
    existing columns and handling renamed or dropped columns.
    - Dropping the original table and renaming the new table.
    - Re-creating any indexes and triggers that existed on the original table.

 3. Dependency Analysis & DAG Construction: Build a Directed Acyclic Graph (DAG)
    using the lowered, squashed operations. Edges are established based on schema
    constraints (e.g., a table containing a foreign key depends on the referenced
    parent table).

 4. Topological Sorting (Execution Order): Traverse the DAG to determine a strict,
    mathematically valid execution order. This guarantees that parent tables are
    created before children, children are dropped before parents, and the database
    never enters a temporarily invalid state.

 5. Transaction Grouping & Pragmas: Group the topologically sorted operations into
    logical transaction blocks to ensure atomicity. Inject required session-level
    instructions, such as toggling `PRAGMA foreign_keys = OFF;` before executing
    12-step recreations, and re-enabling it afterward.
*/
func Plan(schem *schema.Schema, ops []op.Op) ([]op.Op, error) {
	var errs []error
	for _, op := range ops {
		fmt.Printf("%T %+v", op, op)
	}

	loweredOps := []op.Op{}
	opsByTable := groupOpsByTargetTable(ops)
	for tableName, tableOps := range opsByTable {
		if requiresRecreation(tableOps) {
			oldTable := schem.Tables[tableName]
			newTable, err := computeNewState(oldTable, tableOps)
			if err != nil {
				errs = append(errs, errext.UnwrapAll(err)...)
			}
			recreationOps := migrateTable(oldTable, newTable)
			loweredOps = append(loweredOps, recreationOps...)
		} else {
			loweredOps = append(loweredOps, tableOps...)
		}
	}

	return ops, nil
}

func groupOpsByTargetTable(ops []op.Op) map[string][]op.Op {
	grouped := make(map[string][]op.Op)

	for _, o := range ops {
		switch o := o.(type) {
		case *op.AddColumn:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.DropColumn:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.RenameColumn:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.ChangeColumnType:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.AddColumnConstraint:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.DropColumnConstraint:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.AddTableConstraint:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.DropTableConstraint:
			grouped[o.Target.Table.Name] = append(grouped[o.Target.Table.Name], o)
		case *op.AddTable:
			grouped[o.Target.Schema.Name] = append(grouped[o.Target.Schema.Name], o)
		case *op.DropTable:
			grouped[o.Target.Schema.Name] = append(grouped[o.Target.Schema.Name], o)
		case *op.RenameTable:
			grouped[o.Target.Schema.Name] = append(grouped[o.Target.Schema.Name], o)
		default:
			panic("unreachable")
		}
	}

	return grouped
}

func requiresRecreation(ops []op.Op) bool {
	result := false

Loop:
	for _, o := range ops {
		switch o.(type) {
		case *op.AddColumn, *op.DropColumn, *op.RenameColumn:
			continue
		case *op.AddTable, *op.DropTable, *op.RenameTable:
			continue
		case *op.AddColumnConstraint, *op.DropColumnConstraint:
			result = true
			break Loop
		case *op.AddTableConstraint, *op.DropTableConstraint:
			result = true
			break Loop
		case *op.ChangeColumnType:
			result = true
			break Loop
		}
	}

	return result
}

func computeNewState(oldTable *schema.Table, ops []op.Op) (*schema.Table, error) {
	newTable := cloneTable(oldTable)

	for _, o := range ops {
		switch o := o.(type) {
		case *op.AddColumn:
			newTable.Columns[o.Column.GetName().String()] = o.Column
		case *op.DropColumn:
			delete(newTable.Columns, o.Target.Column.GetName().String())
		case *op.RenameColumn:
			newTable.Columns[o.Target.Column.GetName().String()].SetName(o.NewName)
		case *op.AddColumnConstraint:
			newTable.Columns[o.Target.Column.GetName().String()].AddConstraint(o.Constraint)
		case *op.DropColumnConstraint:
			newTable.Columns[o.Target.Column.GetName().String()].DropConstraint(o.Target.Constraint)
		case *op.ChangeColumnType:
			newTable.Columns[o.Target.Column.GetName().String()].SetType(o.NewType)
		case *op.AddTableConstraint:
			newTable.AddConstraint(o.Constraint)
		case *op.DropTableConstraint:
			newTable.DropConstraint(o.Target.Constraint)
		}
	}

	// @TODO(woody): we would probably want to validate the new table
	// but we don't have a table.Validate() function right now
	// if we did have a function like that, it would likely takeover
	// responsibility of wriring up the internal references to columns
	// from fks and unique constraints

	return newTable, nil
}

func cloneTable(src *schema.Table) *schema.Table {
	panic("")
}

func migrateTable(oldTable *schema.Table, newTable *schema.Table) []op.Op {
	ops := []op.Op{}

	return ops
}
