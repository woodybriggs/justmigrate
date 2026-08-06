package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/urfave/cli/v3"

	sqlite "justmigrate/internal/dialects/sqlite"

	"justmigrate/internal/backend/diff"
	"justmigrate/internal/backend/formatter"
	"justmigrate/internal/backend/schema"
	"justmigrate/internal/dialects/sqlite/generator"
	"justmigrate/internal/dialects/sqlite/parser"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/lexer"
	"justmigrate/internal/frontend/report"
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

var (
	databaseURL    string   = ""
	databaseDriver string   = "sqlite3"
	outputFile     *os.File = os.Stdout
	schemaFilePath string   = "schema.sql"
)

func pull(ctx context.Context, cmd *cli.Command) (err error) {

	conn, err := sql.Open(databaseDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("error: unable to connect to the database: %w", err)
	}

	db := sqlite.Sqlite{
		DB:       conn,
		FileName: databaseURL,
	}
	source, err := db.ExportDataDefinitions()
	if err != nil {
		return fmt.Errorf("error: unable to export data defintions from db: %w", err)
	}

	if cmd.String("output-file") != "" {
		outputFile, err = os.OpenFile(cmd.String("output-file"), os.O_RDWR|os.O_CREATE, 0644)
		if err != nil {
			return fmt.Errorf("error: unable to open file for writing %s: %w", cmd.String("output-file"), err)
		}
	}

	fmt.Fprint(outputFile, source)
	defer outputFile.Close()

	return err
}

func push(ctx context.Context, cmd *cli.Command) error {

	conn, err := sql.Open(databaseDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("error: unable to connect to the database: %w", err)
	}

	db := sqlite.Sqlite{
		DB:       conn,
		FileName: databaseURL,
	}
	source, err := db.ExportDataDefinitions()
	if err != nil {
		return fmt.Errorf("error: unable to export data defintions from db: %w", err)
	}

	dbSourceReader := strings.NewReader(source)
	_, srcAst, err := astFromReader(dbSourceReader, databaseURL)
	if err != nil {
		return fmt.Errorf("error: parsing ast from database failed: %w", err)
	}

	schemaFilePath := cmd.String("schema-file")
	schemaFile, err := os.Open(schemaFilePath)
	if err != nil {
		return fmt.Errorf("error: open file failed with err %w", err)
	}

	_, tgtAst, err := astFromReader(schemaFile, schemaFilePath)
	if err != nil {
		return fmt.Errorf("error: parsing ast from file failed: %w", err)
	}

	schemaA, err := schema.SchemaFromAst("main", srcAst)
	if err != nil {
		return fmt.Errorf("error: invalid schema\n%w\n", err)
	}

	schemaB, err := schema.SchemaFromAst("main", tgtAst)
	if err != nil {
		return fmt.Errorf("error: invalid schema\n%w\n", err)
	}

	// perform a "diff" of the two ast and procude a set of transform ops
	ops, err := diff.DiffSchema(schemaA, schemaB)
	if err != nil {
		return fmt.Errorf("error: diff'ing schema failed: %w", err)
	}

	// generate the schema changes needed from the ops
	plan, err := generator.Plan(schemaA, schemaB, ops)
	if err != nil {
		if errs, ok := errors.AsType[interface {
			Error() string
			Unwrap() []error
		}](err); ok {
			for _, report := range errs.Unwrap() {
				fmt.Println(report)
			}
		}
		return err
	}

	stmts, err := generator.Translate(schemaA, plan)
	if err != nil {
		if errs, ok := errors.AsType[interface {
			Error() string
			Unwrap() []error
		}](err); ok {
			for _, report := range errs.Unwrap() {
				fmt.Println(report)
			}
		}
		return err
	}

	builder := strings.Builder{}

	fmtter := generator.NewSqliteFormatter(
		true,
		formatter.NewCoreFormatter(&builder, 80, "[]"),
	)
	fmtter.VisitStatements(stmts)

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("error: unable to obtain transaction handle on database: %w", err)
	}

	_, err = tx.ExecContext(ctx, builder.String())
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error: transaction failed, rolled backed: %w", err)
	}
	tx.Commit()

	fmt.Print(builder.String())

	return nil
}

func main() {

	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "database-url",
				Usage:       "url or filepath to database",
				Destination: &databaseURL,
			},
			&cli.StringFlag{
				Name:        "database-driver",
				Usage:       "name of database driver (sqlite3)",
				Destination: &databaseDriver,
			},
		},
		Commands: []*cli.Command{
			&cli.Command{
				Name:   "pull",
				Usage:  "connect to database and pull existing schema into file",
				Action: pull,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "output-file",
						Value: "schema.sql",
						Usage: "where to save the pulled schema",
					},
				},
			},
			&cli.Command{
				Name:   "push",
				Usage:  "compares current db schema with target schema executes statements to migrate",
				Action: push,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "schema-file",
						Value: "schema.sql",
						Usage: "the target schema data definitions",
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
	return
}
