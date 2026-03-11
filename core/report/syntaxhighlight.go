package report

import (
	"fmt"
	"io"
	"woodybriggs/justmigrate/core/luther"
	"woodybriggs/justmigrate/core/tik"
)

type Theme struct {
	IdentifierColor  Color
	KeywordColor     Color
	PunctuationColor Color
	CommentColor     Color
}

var defaultTheme Theme = Theme{
	IdentifierColor:  Color{r: 139, g: 195, b: 226},
	KeywordColor:     Color{r: 86, g: 156, b: 214},
	CommentColor:     Color{r: 105, g: 153, b: 85},
	PunctuationColor: Color{r: 255, g: 255, b: 255},
}

type Color struct {
	r, g, b, _ byte
}

func syntaxHighlight(w io.Writer, line renderLine, theme *Theme) {

	if theme == nil {
		theme = &defaultTheme
	}

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
		setForegroundColor(w, theme.CommentColor)
		fmt.Fprint(w, line.Content)
		resetColor(w)
		return
	}

	for tok.Kind != tik.TokenKind_EOF {
		fmt.Fprint(w, tok.LeadingTrivia)
		if tok.Kind > tik.TokenKindOffset_Keywords {
			setForegroundColor(w, theme.KeywordColor)
			fmt.Fprint(w, tok.Text)
			resetColor(w)
		} else if tok.Kind == tik.TokenKind_Identifier {
			setForegroundColor(w, theme.IdentifierColor)
			fmt.Fprint(w, tok.Quoted())
			resetColor(w)
		} else if tok.Kind < tik.TokenKindOffset_Atoms {
			setForegroundColor(w, theme.PunctuationColor)
			fmt.Fprint(w, tok.Text)
			resetColor(w)
		} else {
			fmt.Fprint(w, tok.Text)
		}
		setForegroundColor(w, theme.CommentColor)
		fmt.Fprint(w, tok.TrailingTrivia)
		resetColor(w)
		tok = miniLex.NextToken()
	}
}

func setForegroundColor(w io.Writer, c Color) {
	fmt.Fprintf(w, "\x1b[38;2;%d;%d;%dm", c.r, c.g, c.b)
}

func resetColor(w io.Writer) {
	fmt.Fprintf(w, "\x1b[39m")
}
