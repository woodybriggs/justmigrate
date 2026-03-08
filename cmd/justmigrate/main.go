package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"woodybriggs/justmigrate/core/ast"
	"woodybriggs/justmigrate/core/diff"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/database"
	sqlitegen "woodybriggs/justmigrate/dialects/sqlite/generator"
	sqliteparser "woodybriggs/justmigrate/dialects/sqlite/parser"

	_ "github.com/mattn/go-sqlite3"
)

var (
	ErrInvalidNode = errors.New("invalid ast node")
)

var (
	ErrParserErrors = errors.New("parser has errors")
)

func assert(cond bool, err error) {
	if !cond {
		panic(err)
	}
}

type Database interface {
	Url() string
	ExportDataDefinitions() (string, error)
}

func ShowErrors(errors []report.Report, w io.Writer) {
	errorRenderer := report.Renderer{}
	for _, report := range errors {
		w.Write([]byte(errorRenderer.Render(report)))
	}
}

func ShowWarnings(warnings []report.Report, w io.Writer) {
	renderer := report.Renderer{}
	for _, report := range warnings {
		w.Write([]byte(renderer.Render(report)))
	}
}

func AstFromDatabase(database Database) (luther.SourceCode, []ast.Statement, error) {
	source, err := database.ExportDataDefinitions()
	if err != nil {
		return luther.SourceCode{}, nil, err
	}

	lexer := luther.NewLexer(
		luther.SourceCode{
			FileName: database.Url(),
			Raw:      []rune(source),
		},
	)

	parser := sqliteparser.NewSqliteParser(lexer)

	nodes := parser.Statements()
	errors := parser.Errors()
	if len(errors) > 0 {
		ShowErrors(errors, os.Stderr)
		return parser.Current().SourceCode, nil, ErrParserErrors
	}

	// warnings := slices.Collect(maps.Values(parser.Warnings))
	// if len(warnings) > 0 {
	// 	ShowWarnings(warnings, os.Stderr)
	// }

	return parser.Parser.Current().SourceCode, nodes, nil
}

func AstFromFile(file *os.File) (luther.SourceCode, []ast.Statement, error) {
	lexer, err := luther.NewLexerFromFile(file)
	if err != nil {
		return lexer.SourceCode, nil, err
	}

	parser := sqliteparser.NewSqliteParser(lexer)

	nodes := parser.Statements()
	errors := parser.Errors()

	if len(errors) > 0 {
		ShowErrors(errors, os.Stderr)
		return lexer.SourceCode, nil, ErrParserErrors
	}

	// warnings := slices.Collect(maps.Values(parser.Warnings))
	// if len(warnings) > 0 {
	// 	ShowWarnings(warnings, os.Stderr)
	// }

	return lexer.SourceCode, nodes, nil
}

func main() {

	var err error

	databaseURL := "resources/database.db"

	conn, err := sql.Open("sqlite3", databaseURL)
	if err != nil {
		log.Panicln(err)
	}

	db := &database.Sqlite{DB: conn, FileName: databaseURL}

	fileName := "resources/schema.sql"
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file failed with err %v", err)
		os.Exit(1)
	}

	_, dstAst, err := AstFromFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ast from file failed with err %v", err)
		os.Exit(1)
	}

	_, srcAst, err := AstFromDatabase(db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ast from database failed with err %v", err)
		os.Exit(1)
	}

	differ := diff.Diff{}

	ops, err := differ.DiffSchema(srcAst, dstAst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diff schema failed with err %v", err)
		os.Exit(1)
	}

	gen := sqlitegen.SqliteGenerator{}

	ops, err = gen.Plan(dstAst, ops)

	type multiError interface {
		Unwrap() []error
	}
	if err != nil {
		if errs, ok := errors.AsType[*sqlitegen.MissingColumnsErr](err); ok {
			for _, err := range errs.Unwrap() {
				fmt.Println(err)
			}
		}
		if errs, ok := errors.AsType[*sqlitegen.ErrSchemaResolutionFailed](err); ok {
			for _, err := range errs.Unwrap() {
				fmt.Println(err)
			}
		}
	}

	for _, op := range ops {
		fmt.Printf("%T, %+v\n", op, op)
	}
}
