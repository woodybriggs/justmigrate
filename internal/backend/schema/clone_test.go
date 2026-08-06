package schema_test

import (
	"testing"
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/dialects/sqlite/parser"
	"justmigrate/internal/frontend/lexer"
)

func parseTable(t *testing.T, name, source string) *schema.Table {
	t.Helper()
	l := lexer.NewLexer(lexer.SourceCode{
		FileName: name,
		Raw:      []rune(source),
	})
	p := parser.NewSqliteParser(l)
	createTable := p.CreateTableStatement(false)
	if len(p.ErrorsAsErrorSlice()) > 0 {
		t.Fatalf("parse errors: %v", p.ErrorsAsErrorSlice())
	}
	table, err := schema.TableFromAst(createTable)
	if err != nil {
		t.Fatalf("TableFromAst: %v", err)
	}
	return table
}

func parseSchema(t *testing.T, name, source string) *schema.Schema {
	t.Helper()
	l := lexer.NewLexer(lexer.SourceCode{
		FileName: name,
		Raw:      []rune(source),
	})
	p := parser.NewSqliteParser(l)
	stmts := p.Statements()
	s, err := schema.SchemaFromAst("main", stmts)
	if err != nil {
		t.Fatalf("SchemaFromAst: %v", err)
	}
	return s
}

func Test_CloneTable_ColumnsAreIndependent(t *testing.T) {
	orig := parseTable(t, "test", `create table users(
		id TEXT not null,
		email TEXT unique not null
	)`)

	clone := orig.Clone()

	origCol := orig.Columns["email"]
	cloneCol := clone.Columns["email"]

	if origCol == cloneCol {
		t.Fatal("columns should be different objects")
	}

	if origCol.GetName().String() != cloneCol.GetName().String() {
		t.Fatalf("column names should match: got %q, want %q", cloneCol.GetName().String(), origCol.GetName().String())
	}
}

func Test_CloneTable_ConstraintsAreIndependent(t *testing.T) {
	orig := parseTable(t, "test", `create table users(
		id TEXT not null,
		name TEXT,
		primary key (id)
	)`)

	clone := orig.Clone()

	if orig.Constraints.PK == clone.Constraints.PK {
		t.Fatal("PK should be different objects")
	}

	if orig.Constraints.PK.ResolvedColumns[0] == clone.Constraints.PK.ResolvedColumns[0] {
		t.Fatal("PK ResolvedColumns should point to cloned columns")
	}
}

func Test_CloneTable_PKResolvedColumnsPointToClonedColumns(t *testing.T) {
	orig := parseTable(t, "test", `create table users(
		id TEXT not null,
		name TEXT,
		primary key (id)
	)`)

	clone := orig.Clone()

	origPKCol := orig.Constraints.PK.ResolvedColumns[0]
	clonePKCol := clone.Constraints.PK.ResolvedColumns[0]

	if origPKCol == clonePKCol {
		t.Fatal("PK ResolvedColumns should point to different column objects")
	}

	if origPKCol.GetName().String() != clonePKCol.GetName().String() {
		t.Fatalf("PK column names should match: got %q, want %q", clonePKCol.GetName().String(), origPKCol.GetName().String())
	}

	origIdCol := orig.Columns["id"]
	cloneIdCol := clone.Columns["id"]

	if origPKCol != origIdCol {
		t.Fatal("original PK should reference original id column")
	}

	if clonePKCol != cloneIdCol {
		t.Fatal("cloned PK should reference cloned id column")
	}
}

func Test_CloneTable_FKFromColumnsPointToClonedColumns(t *testing.T) {
	s := parseSchema(t, "test", `
		create table users(id TEXT not null);
		create table orders(
			id TEXT not null,
			user_id TEXT not null,
			foreign key (user_id) references users(id)
		)`)

	orig := s.Tables["orders"]
	clone := orig.Clone()

	if len(orig.Constraints.FKs) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(orig.Constraints.FKs))
	}

	origFK := orig.Constraints.FKs[0]
	cloneFK := clone.Constraints.FKs[0]

	if origFK == cloneFK {
		t.Fatal("FK should be different objects")
	}

	if len(origFK.FromColumns) != 1 || len(cloneFK.FromColumns) != 1 {
		t.Fatalf("expected 1 FromColumn each, got %d and %d", len(origFK.FromColumns), len(cloneFK.FromColumns))
	}

	if origFK.FromColumns[0] == cloneFK.FromColumns[0] {
		t.Fatal("FK FromColumns should point to cloned columns")
	}

	origUserIdCol := orig.Columns["user_id"]
	cloneUserIdCol := clone.Columns["user_id"]

	if origFK.FromColumns[0] != origUserIdCol {
		t.Fatal("original FK FromColumns should reference original column")
	}

	if cloneFK.FromColumns[0] != cloneUserIdCol {
		t.Fatal("cloned FK FromColumns should reference cloned column")
	}

	if cloneFK.ToTable != s.Tables["users"] {
		t.Fatal("FK ToTable should still reference the original users table")
	}
}

func Test_CloneTable_ModifyingCloneDoesNotAffectOriginal(t *testing.T) {
	orig := parseTable(t, "test", `create table users(
		id TEXT not null,
		email TEXT not null
	)`)

	clone := orig.Clone()

	delete(clone.Columns, "email")

	if _, ok := orig.Columns["email"]; !ok {
		t.Fatal("deleting from clone should not affect original")
	}

	clone.Name = "modified"
	if orig.Name == "modified" {
		t.Fatal("modifying clone name should not affect original")
	}
}

func Test_CloneTable_SelfReferentialFK(t *testing.T) {
	s := parseSchema(t, "test", `create table categories(
		id TEXT not null,
		parent_id TEXT,
		foreign key (parent_id) references categories(id)
	)`)

	orig := s.Tables["categories"]
	clone := orig.Clone()

	if len(clone.Constraints.FKs) != 1 {
		t.Fatalf("expected 1 FK, got %d", len(clone.Constraints.FKs))
	}

	fk := clone.Constraints.FKs[0]

	if fk.FromTable != clone {
		t.Fatal("self-referential FK FromTable should point to cloned table")
	}

	if fk.ToTable != orig {
		t.Fatal("self-referential FK ToTable should still point to original table")
	}

	if fk.FromColumns[0] != clone.Columns["parent_id"] {
		t.Fatal("self-referential FK FromColumns should reference cloned column")
	}
}
