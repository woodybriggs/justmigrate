package sqlite

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/tik"
	"woodybriggs/justmigrate/formatter"
)

func makeParser(input string) *SqliteParser {

	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		panic("unable to get caller info")
	}
	funcInfo := runtime.FuncForPC(pc)
	file, _ := funcInfo.FileLine(pc)

	lex := luther.NewLexer(luther.SourceCode{
		FileName: fmt.Sprintf("%s/%s", file, funcInfo.Name()),
		Raw:      []rune(input),
	})

	return NewSqliteParser(lex)
}

func makeFormatter() (*strings.Builder, *formatter.CoreFormatter) {
	sb := &strings.Builder{}
	return sb, formatter.NewCoreFormatter(sb, 300, "")
}

func TestCreateTable(t *testing.T) {
	expectedStr := "CREATE TABLE IF NOT EXISTS users (\n\tid integer PRIMARY KEY AUTOINCREMENT\n)"
	parser := makeParser(expectedStr)

	parsedAst := parser.Statement()
	expectedAst := &ast.CreateTable{
		TableIdentifier: &ast.CatalogObjectIdentifier{
			ObjectName: ast.Identifier{Text: "users"},
		},
		TableDefinition: &ast.TableDefinition{
			ColumnDefinitions: []ast.ColumnDefinition{
				{
					ColumnName: ast.Identifier{Text: "id"},
					TypeName: &ast.TypeName{
						Name: ast.Identifier{
							Text: "integer",
						},
					},
					ColumnConstraints: []ast.ColumnConstraint{
						&ast.ColumnConstraint_PrimaryKey{
							AutoIncrement: &ast.Keyword{Kind: tik.TokenKind_Keyword_AUTOINCREMENT},
						},
					},
				},
			},
		},
	}

	if !parsedAst.Eq(expectedAst) {
		t.Fail()
	}
}

func TestParseIdentifier(t *testing.T) {
	parser := makeParser("user_id [user_id] `user_id` \"user_id\"")

	for !parser.EndOfFile() {
		ident := parser.Identifier()
		if ident.Text != "user_id" {
			t.FailNow()
		}
	}
}
