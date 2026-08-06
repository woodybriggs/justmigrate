package parser

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"justmigrate/internal/frontend/ast"
	"justmigrate/internal/frontend/token"
)

func (p *SqliteParser) Statements() []ast.Statement {
	statements := []ast.Statement{}

	loopCount := 0
	for !p.EndOfFile() && loopCount < 1024 {
		func(loopCount int) {
			defer func() {
				if r := recover(); r != nil {
					if err, isErr := r.(error); isErr && errors.Is(err, ErrNotImplemented) {
						debug.PrintStack()
						os.Exit(2)
					}
					p.Synchronize([]token.TokenKind{';'})
				}
			}()

			statement := p.Statement()
			statements = append(statements, statement)

			// if this fails/panics, the defer block above handles it too.
			p.Expect(';')
		}(loopCount)
		loopCount++
	}

	return statements
}

func (p *SqliteParser) Statement() ast.Statement {
	p.PushParseContext("statement")
	defer p.PopParseContext()

	switch p.Current().Kind {
	case token.TokenKind_Keyword_CREATE:
		return p.CreateStatement()
	default:
		fmt.Fprintf(os.Stderr, "unhandled statement %v", p.Current().Text)
		os.Exit(1)
		panic(ErrNotImplemented)
	}
}
