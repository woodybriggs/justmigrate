package report

import (
	"fmt"
	"io"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/tik"
)

type Color struct {
	r, g, b, _ byte
}

func syntaxHighlight(w io.Writer, line renderLine) {
	commentColor := Color{r: 0, g: 150, b: 0}
	identifierColor := Color{r: 150, g: 250, b: 250}
	keywordColor := Color{r: 0, g: 75, b: 200}
	punctuationColor := Color{r: 255, g: 255, b: 255}

	if !line.IsSrc {
		fmt.Fprint(w, line.Content)
		return
	}

	source := luther.SourceCode{
		FileName: "n/a",
		Raw:      []rune(line.Content),
	}

	miniLex := luther.Lexer{
		SourceCode: source,
		LexerData: luther.LexerData{
			Row: 1,
		},
	}

	tok := miniLex.NextToken()
	if tok.Kind == tik.TokenKind_EOF && len(line.Content) != 0 {
		// this must be a line comment
		setForegroundColor(w, commentColor)
		fmt.Fprint(w, line.Content)
		resetColor(w)
		return
	}

	for tok.Kind != tik.TokenKind_EOF {
		fmt.Fprint(w, tok.LeadingTrivia)
		if tok.Kind > tik.TokenKindOffset_Keywords {
			setForegroundColor(w, keywordColor)
			fmt.Fprint(w, tok.Text)
			resetColor(w)
		} else if tok.Kind == tik.TokenKind_Identifier {
			setForegroundColor(w, identifierColor)
			fmt.Fprint(w, tok.Quoted())
			resetColor(w)
		} else if tok.Kind < tik.TokenKindOffset_Atoms {
			setForegroundColor(w, punctuationColor)
			fmt.Fprint(w, tok.Text)
			resetColor(w)
		} else {
			fmt.Fprint(w, tok.Text)
		}
		setForegroundColor(w, commentColor)
		fmt.Fprint(w, tok.TrailingTrivia)
		resetColor(w)
		tok = miniLex.NextToken()
	}

	return
}

func setForegroundColor(w io.Writer, c Color) {
	fmt.Fprintf(w, "\x1b[38;2;%d;%d;%dm", c.r, c.g, c.b)
}

func resetColor(w io.Writer) {
	fmt.Fprintf(w, "\x1b[39m")
}
