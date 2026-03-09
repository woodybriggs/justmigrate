package generator

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/diff"
	"woodybriggs/justmigrate/core/formatter"
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
func (gen *SqliteFormatter) Plan(src, tgt []ast.Statement, ops []diff.Op) ([]diff.Op, error) {
	var errs []error
	srcGraph, err := NewSchemaGraphFromStatements(src)
	if err != nil {
		if er, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range er.Unwrap() {
				errs = append(errs, e)
			}
		}
	}
	tgtGraph, err := NewSchemaGraphFromStatements(tgt)
	if err != nil {
		if er, ok := err.(interface{ Unwrap() []error }); ok {
			for _, e := range er.Unwrap() {
				errs = append(errs, e)
			}
		}
	}
	_ = tgtGraph

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	for _, op := range ops {
		fmt.Printf("%T %+v", op, op)
	}

	var plan []diff.Op
	for _, op := range ops {
		switch o := op.(type) {
		case *diff.DelColOp:
			// can we be certain about this lookup?, given that the ops were generated from the schemas

			// if deleting the column will break a foreign key somewhere
			// we need to drop the fk constraint in that table
			// for now I think we push a generic "drop constraint op"
			// and we will have to do at least 2 passes over the plan
			// to ensure that the plan is suitable for the dialect in this case 'sqlite'

			ogTable := srcGraph.Tables[o.Table.ObjectName.Text]
			childTables := srcGraph.Columns[ogTable.Name][o.Col.Text].DependantTables
			for _, child := range childTables {
				// for each dependant child
				_ = child
			}

			newTable := ast.Copy(ogTable.CreateTable).(*ast.CreateTable)
			dropColumn(newTable, o.Col)

			fmtter := NewSqliteFormatter(false, formatter.NewCoreFormatter(os.Stdout, 80, "\"\""))
			fmtter.VisitStatements([]ast.Statement{newTable})
		case *diff.ChangeColTypeOp:
			// These operations are not natively supported and require lowering to a
			// full table recreation.
			// The `lowerTableRecreation` helper would generate the sequence:
			//  - CREATE new table
			//  - INSERT INTO new_table SELECT ... FROM old_table
			//  - DROP old_table
			//  - RENAME new_table
			//  - Re-create indexes and triggers
			// recreateOps, err := gen.lowerTableRecreation(o.TableName, statements, schemaGraph)
			// if err != nil {
			// 	return nil, err
			// }
			// plan = append(plan, recreateOps...)
			_ = o // Avoid unused variable error for this example

		default:
			// By default, assume the operation is natively supported (e.g., CreateTable,
			// AddColumn, DropTable). These can be added directly to the plan.
			// The final sorting step will handle their execution order.
			plan = append(plan, o)
		}
	}

	// 3. Sort the final plan to respect dependencies (e.g., using a topological sort).
	// sortedPlan, err := sortOps(plan, schemaGraph)

	// 4. Add final pragmas.
	// plan = append(plan, &PragmaOp{Key: "foreign_keys", Value: "ON"})
	// plan = append(plan, &PragmaOp{Key: "foreign_key_check", Value: ""})

	return plan, nil
}

func dropColumn(table *ast.CreateTable, colName *ast.Identifier) {
	indexOfCol := slices.IndexFunc(table.TableDefinition.ColumnDefinitions, func(col ast.ColumnDefinition) bool {
		return col.ColumnName.Eq(colName.AsExpr())
	})

	if indexOfCol == -1 {
		return
	}

	table.TableDefinition.ColumnDefinitions = append(
		table.TableDefinition.ColumnDefinitions[:indexOfCol],
		table.TableDefinition.ColumnDefinitions[indexOfCol+1:]...,
	)
}
