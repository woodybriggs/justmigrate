package luther

import (
	"fmt"
	"os"
	"testing"
	"woodybriggs/justmigrate/core/tik"
)

type Case struct {
	filename     string
	input        string
	expectedKind tik.TokenKind
	expectedText string
}

func TestNumericLiteral(t *testing.T) {

	cases := []Case{
		{input: "1", expectedText: "1", expectedKind: tik.TokenKind_IntegerNumericLiteral},
		{input: "1.1", expectedText: "1.1", expectedKind: tik.TokenKind_FloatNumericLiteral},
		{input: "1.1e+7", expectedText: "1.1e+7", expectedKind: tik.TokenKind_FloatNumericLiteral},
		{input: "1.1e-7", expectedText: "1.1e-7", expectedKind: tik.TokenKind_FloatNumericLiteral},
		{input: ".1e7", expectedText: ".1e7", expectedKind: tik.TokenKind_FloatNumericLiteral},
	}

	for _, cas := range cases {
		lex := NewLexer(SourceCode{FileName: cas.expectedText, Raw: []rune(cas.input)})
		result := lex.NextToken()
		if result.Kind != cas.expectedKind {
			fmt.Fprintf(os.Stderr, "lexing '%s' expected kind '%v' got kind '%v'\n", cas.input, cas.expectedKind.DebugString(), result.Kind.DebugString())
			t.Fail()
		}
		if cas.expectedText != result.Text {
			fmt.Fprintf(os.Stderr, "lexing '%s' expected text '%s' got text '%s'\n", cas.input, cas.expectedText, result.Text)
			t.Fail()
		}
	}
}
