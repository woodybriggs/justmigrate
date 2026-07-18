package schema_test

import (
	"testing"
	"woodybriggs/justmigrate/backend/schema"
	"woodybriggs/justmigrate/dialects/sqlite/parser"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/lexer"
	"woodybriggs/justmigrate/frontend/token"
)

func Test_PrimaryKeyFromTableConstraintAst(t *testing.T) {
	source := "create table users(\n" +
		"    id TEXT not null,\n" +
		"    email TEXT unique not null,\n" +
		"    primary key (654)\n" +
		")"

	l := lexer.NewLexer(
		lexer.SourceCode{
			FileName: "Test_PrimaryKeyFromTableConstraintAst",
			Raw:      []rune(source),
		},
	)

	p := parser.NewSqliteParser(l)

	createTable := p.CreateTableStatement(false)
	if len(p.ErrorsAsErrorSlice()) > 0 {
		t.FailNow()
	}

	_, err := schema.TableFromAst(createTable)
	if err != nil {
		t.Fatalf("\n%v\n", err)
	}
}

func Test_PrimaryKeyFromColumnConstraintAst(t *testing.T) {
	source := "PRIMARY KEY"

	l := lexer.NewLexer(
		lexer.SourceCode{
			FileName: "Test_PrimaryKeyFromTableConstraintAst",
			Raw:      []rune(source),
		},
	)

	p := parser.NewSqliteParser(l)

	pk := p.ColumnConstraint_PrimaryKey(nil)
	if len(p.ErrorsAsErrorSlice()) > 0 {
		t.FailNow()
	}

	columnName := ast.MakeIdentifier(token.Token{Text: "unnamed", Kind: token.TokenKind_Identifier})
	column := schema.Column{
		Name: columnName.String(),
		ColumnDefinition: ast.MakeColumnDefinition(
			*columnName,
			ast.MakeTypeName(
				*ast.MakeIdentifier(token.Token{Text: "TEXT", Kind: token.TokenKind_Identifier}),
				nil,
				nil,
			),
			[]ast.ColumnConstraint{
				ast.MakeColumnConstraintPrimaryKey(nil, ast.Keyword{}, ast.Keyword{}, nil, nil, nil),
			},
		),
	}

	err := schema.PrimaryKeyFromColumnConstraintAst(
		&column,
		&column.ColumnConstraints,
		pk,
	)

	if err != nil {
		t.FailNow()
	}

	if column.ColumnConstraints.PK == nil {
		t.Fatalf("ColumnConstraints should have a PrimaryKey set here")
	}
}
