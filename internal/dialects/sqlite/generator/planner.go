package generator

import (
	"errors"
	"fmt"
	"justmigrate/internal/backend/op"
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/sliceext"
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
func Plan(src, tgt *schema.Schema, ops []op.Op) ([]op.Op, error) {
	var errs []error
	loweredOps := []op.Op{}
	opsByTable := groupOpsByTargetTable(ops)
	for tableName, tableOps := range opsByTable {

		// skip schema related ops such as
		// AddTable, DropTable
		if tableName == src.Name {
			loweredOps = append(loweredOps, tableOps...)
			continue
		}

		// we can skip table ops that don't need full table recreation
		if !requiresRecreation(tableOps) {
			loweredOps = append(loweredOps, tableOps...)
			continue
		}

		oldTable := src.Tables[tableName]
		var newTableName string = tableName

		renameOp, found := sliceext.FindFunc(tableOps, func(o op.Op) bool {
			_, ok := o.(*op.RenameTable)
			return ok
		})
		if found {
			newTableName = renameOp.(*op.RenameTable).NewName.ObjectName.String()
		}
		newTable := tgt.Tables[newTableName]
		recreationOps := createMigrateDropRename(src, oldTable, newTable, tableOps)
		loweredOps = append(loweredOps, recreationOps...)
	}

	return loweredOps, errors.Join(errs...)
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
	newTable := oldTable.Clone()

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
	if err := newTable.ResolveInternalReferences(); err != nil {
		return nil, err
	}

	return newTable, nil
}

/*
migrateTable generates the `INSERT INTO new_table (col1) SELECT expr1 FROM old_table`
statement during a 12-step SQLite table recreation.

To accurately migrate data, the algorithm must build two perfectly aligned lists
(destination columns and source expressions) by inferring the relationship between
the old table, the new table, and the operations applied.

Operation Mappings:
-------------------------------------------------------------------------
| Operation     | Destination (INSERT) | Source (SELECT)                |
|---------------|----------------------|--------------------------------|
| Unchanged     | col_name             | col_name                       |
| Drop Column   | N/A (Excluded)       | N/A (Excluded)                 |
| Add Column    | new_col_name         | NULL or Default Value          |
| Rename Column | new_col_name         | old_col_name                   |
| Modify Type   | col_name             | col_name (SQLite auto-casts)   |
-------------------------------------------------------------------------

The Inference Algorithm:
Drive the logic by iterating over the `newTable.Columns` to determine exactly
what needs to be inserted.

 1. Initialize Lists: Create empty string slices for `insertCols` and `selectExprs`.
 2. Iterate Target: Loop through every column in `newTable.Columns`.
 3. Check Renames First: Scan `ops` for an `op.RenameColumn` where the
    `NewName` matches the current column. If found, append the new name to
    `insertCols`, append the *old* name to `selectExprs`, and move to the next column.
 4. Check Unchanged/Modified: Look for the column name in `oldTable.Columns`.
    If it exists (meaning the column was kept, even if type/constraints changed),
    append the column name to both `insertCols` and `selectExprs`.
 5. Handle Added Columns: If the column is not a rename and not in `oldTable`,
    it is newly added. Append the new name to `insertCols`. For `selectExprs`:
    - If `newTable` defines a default value, append that default expression.
    - If there is no default value, append `NULL`.

Critical Considerations:
  - The NOT NULL Trap: If a user adds a column with a `NOT NULL` constraint but
    fails to provide a `DEFAULT` value, mapping the source to `NULL` will cause
    the `INSERT` to fail. This must be validated and caught prior to this step
    (e.g., during AST parsing or initial operation validation).
  - Implicit Drops: Dropped columns are handled implicitly. Because the loop is
    driven by `newTable.Columns`, any column present in `oldTable` but missing
    from `newTable` is naturally skipped, leaving the dropped data behind.
  - Quoting: Always escape column names in the generated SQL (e.g., `"my_col"`)
    to prevent syntax errors from reserved keywords.
*/
func createMigrateDropRename(schem *schema.Schema, old, new *schema.Table, ops []op.Op) []op.Op {

	tmpTable := new.Clone()
	tmpName := fmt.Sprintf("tmpnew_%s", new.Name)
	tmpTable.Name = tmpName

	return []op.Op{
		&op.AddTable{
			Target: op.TargetSchema{
				Schema: schem,
			},
			Table: tmpTable,
		},
		&op.MigrateData{
			SrcTarget: op.TargetTable{
				Schema: schem,
				Table:  old,
			},
			DstTarget: op.TargetTable{
				Schema: schem,
				Table:  tmpTable,
			},
			Ops: ops,
		},
		&op.DropTable{
			Target: op.TargetTable{
				Schema: schem,
				Table:  old,
			},
		},
		&op.RenameTable{
			Target: op.TargetTable{
				Schema: schem,
				Table:  tmpTable,
			},
			NewName: old.Node.TableIdentifier,
		},
	}
}
