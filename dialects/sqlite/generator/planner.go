package generator

import (
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/diff"
)

// Plan takes a first pass of "dumb" operations (create table, drop table, etc.)
// and transforms them into a detailed, ordered, and valid execution plan for SQLite.
//
// This process is particularly complex for SQLite due to its limited `ALTER TABLE`
// support, which often necessitates a full table recreation for many common modifications.
//
// The planner's responsibilities include:
//
//  1. Schema Graph Construction: Before planning, the planner must have access to a
//     graph representation of the database schema, including tables, columns,
//     foreign keys, indexes, and triggers. This graph is essential for
//     dependency analysis.
//
//  2. Operation Validation: Determine if each requested operation is natively
//     supported by the SQLite dialect. For example, `ADD COLUMN` is supported,
//     but `DROP COLUMN` is only supported in recent SQLite versions and may need
//     to be lowered.
//
//  3. Operation Lowering: Convert high-level, unsupported operations into a
//     sequence of simpler, supported SQLite operations. The primary example is
//     the "12-step" table recreation strategy for changes like dropping a column,
//     altering a column's type, or adding a foreign key to an existing table.
//     This involves:
//     - Creating a new table with the desired schema.
//     - Generating an `INSERT INTO ... SELECT ...` statement to migrate data
//     from the old table to the new one, correctly mapping columns.
//     - Dropping the original table.
//     - Renaming the new table to the original's name.
//     - Re-creating any indexes and triggers that existed on the original table.
//
//  4. Dependency Analysis: Identify and resolve dependencies between operations
//     using the schema graph. For instance, a table cannot be dropped if it is
//     referenced by a foreign key in another table. The planner must ensure
//     the foreign key constraint is dropped first or that the referencing table
//     is also part of a recreation plan.
//
//  5. Execution Ordering & Pragmas: Arrange the final sequence of operations
//     into an executable order that satisfies all dependencies. This also involves
//     injecting necessary session pragmas like `PRAGMA foreign_keys = OFF;` at the
//     start and `PRAGMA foreign_keys = ON;` at the end of the migration.
//
//  6. Transaction Grouping: Group related sequences of operations (like the
//     entire table recreation process) into logical units that should be
//     executed within a single transaction to ensure atomicity.
func (gen *SqliteGenerator) Plan(statements []ast.Statement, ops []diff.Op) ([]diff.Op, error) {

	schemaGraph := NewSchemaGraph()

	for _, stmt := range statements {
		switch typ := stmt.(type) {
		case *ast.CreateTable:
			err := schemaGraph.AddTable(typ)
			if err != nil {
				return nil, err
			}
		default:
			continue
		}
	}

	err := schemaGraph.Resolve()
	if err != nil {
		return nil, err
	}

	_, err = schemaGraph.Sort(statements)

	return nil, nil
}
