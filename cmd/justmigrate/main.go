package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	schema "woodybriggs/justmigrate/backend/schema"
	"woodybriggs/justmigrate/dialects/sqlite/parser"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/lexer"
	"woodybriggs/justmigrate/frontend/report"
)

var (
	ErrInvalidNode = errors.New("invalid ast node")
)

type ParserErrors struct {
	Errs []error
}

func (e *ParserErrors) Error() string {
	return fmt.Sprintf("parser has %d errors", len(e.Errs))
}

func (e *ParserErrors) Unwrap() []error {
	return e.Errs
}

func assert(cond bool, err error) {
	if !cond {
		panic(err)
	}
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

func astFromReader(r io.Reader, sourceName string) (lexer.SourceCode, []ast.Statement, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return lexer.SourceCode{}, nil, err
	}

	sourceCode := lexer.SourceCode{
		FileName: sourceName,
		Raw:      []rune(string(data)),
	}

	lexer := lexer.NewLexer(sourceCode)
	parser := parser.NewSqliteParser(lexer)

	statements := parser.Statements()
	errors := parser.ErrorsAsErrorSlice()
	if len(errors) > 0 {
		ShowErrors(parser.ErrorsAsReportSlice(), os.Stderr)

		return sourceCode, nil, &ParserErrors{
			Errs: errors,
		}
	}

	return sourceCode, statements, nil
}

func main() {

	var err error

	// load the current state of the connected database into ast
	// databaseURL := "resources/database.db"
	// conn, err := sql.Open("sqlite3", databaseURL)
	// if err != nil {
	// 	log.Panicln(err)
	// }

	// db := &sqlite.Sqlite{DB: conn, FileName: databaseURL}
	// source, err := db.ExportDataDefinitions()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "unable to export data defintions from db %v", err)
	// 	os.Exit(1)
	// }

	// dbSourceReader := strings.NewReader(source)
	// _, srcAst, err := astFromReader(dbSourceReader, databaseURL)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "ast from database failed with err %v", err)
	// 	os.Exit(1)
	// }

	// load the current state of the target schema file into ast
	fileNameA := "./resources/two/a.sql"
	fileA, err := os.Open(fileNameA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file failed with err %v\n", err)
		os.Exit(1)
	}

	_, srcAst, err := astFromReader(fileA, fileNameA)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ast from file failed with err %v\n", err)
		os.Exit(1)
	}

	fileNameB := "./resources/two/b.sql"
	fileB, err := os.Open(fileNameB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open file failed with err %v\n", err)
		os.Exit(1)
	}

	_, tgtAst, err := astFromReader(fileB, fileNameB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ast from file failed with err %v\n", err)
		os.Exit(1)
	}

	schemaA, err := schema.SchemaFromAst(srcAst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid schema\n%v\n", err)
		os.Exit(1)
	}

	schemaB, err := schema.SchemaFromAst(tgtAst)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid schema\n%v\n", err)
		os.Exit(1)
	}

	_, _ = schemaA, schemaB

	// perform a "diff" of the two ast and procude a set of transform ops
	// differ := diff.Diff{}
	// ops, err := differ.DiffSchema(srcAst, tgtAst)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "diff schema failed with err %v\n", err)
	// 	os.Exit(1)
	// }

	// // validate, reoreder and optimize the ops
	// {

	// }

	// // generate the schema changes needed from the ops
	// gen := generator.SqliteFormatter{}
	// ops, err = gen.Plan(srcAst, tgtAst, ops)
	// type multiError interface {
	// 	Error() string
	// 	Unwrap() []error
	// }
	// if err != nil {
	// 	if errs, ok := errors.AsType[multiError](err); ok {
	// 		for _, report := range errs.Unwrap() {
	// 			fmt.Println(report)
	// 		}
	// 	}
	// }

	// for _, op := range ops {
	// 	fmt.Printf("%T, %+v\n", op, op)
	// }
}
