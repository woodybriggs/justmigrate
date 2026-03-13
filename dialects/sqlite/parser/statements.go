package parser

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"woodybriggs/justmigrate/frontend/ast"
	"woodybriggs/justmigrate/frontend/token"
)

func (p *SqliteParser) Statements() []ast.Statement {
	statements := []ast.Statement{}

	for !p.EndOfFile() {
		func() {
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
		}()
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
		fmt.Fprintf(os.Stderr, "unhandled statement")
		os.Exit(1)
		panic(ErrNotImplemented)
	}
}
