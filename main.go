package main

import (
	"fmt"
	"os"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/report"
	"woodybriggs/justmigrate/dialects/sqlite/parser"
	"woodybriggs/justmigrate/parser"
)

func main() {

	filename := "./resources/test.sql"

	file, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	tokenizer, err := luther.NewLexerFromFile(file)
	if err != nil {
		panic(err)
	}

	parser := sqlite.NewSqliteParser(tokenizer)
	parser.Statements()

	renderer := report.Renderer{}

	for _, report := range parser.Errors() {
		renderer.Render(report)
	}
}
