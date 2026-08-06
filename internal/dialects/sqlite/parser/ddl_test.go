package parser_test

import (
	"testing"
	"justmigrate/internal/dialects/sqlite/parser"
	"justmigrate/internal/frontend/lexer"
)

func Test_CreateTable_Minimal(t *testing.T) {
	sql := "-- Standard creation with minimal constraints\n" +
		"CREATE TABLE users (id INTEGER);"

	l := lexer.NewLexer(lexer.SourceCode{
		FileName: "Test_CreateTable_Minimal",
		Raw:      []rune(sql),
	})

	p := parser.NewSqliteParser(l)

	stmts := p.Statements()

	if len(stmts) != 1 {
		t.FailNow()
	}

}

/*

-- Standard creation with minimal constraints
CREATE TABLE users (id INTEGER);

-- IF NOT EXISTS, multiple constraints, and SQLite 3.37+ STRICT mode
CREATE TABLE IF NOT EXISTS employees (
	emp_id INTEGER PRIMARY KEY AUTOINCREMENT,
	first_name TEXT NOT NULL,
	last_name TEXT,
	email TEXT UNIQUE,
	hire_date DATE DEFAULT CURRENT_DATE,
	salary REAL CHECK(salary > 0)
) STRICT;

-- Foreign keys, COLLATE, and WITHOUT ROWID optimization
CREATE TABLE orders (
	order_id INTEGER PRIMARY KEY,
	user_id INTEGER,
	status TEXT COLLATE NOCASE,
	FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE SET NULL
) WITHOUT ROWID;

-- Escaped identifiers
CREATE TABLE "table with spaces" (
	[column name] TEXT,
	`another column` INT
);

*/
