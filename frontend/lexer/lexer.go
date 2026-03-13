package lexer

import (
	"io"
	"os"
	"strings"
	"unicode"
	"woodybriggs/justmigrate/frontend/token"
)

type SourceCode struct {
	FileName string
	Raw      []rune
}

type LexerData struct {
	Cur int
	Bol int
	Row int
}

type Lexer struct {
	SourceCode
	LexerData
}

func (l *Lexer) Clone() *Lexer {
	return &Lexer{
		SourceCode: l.SourceCode,
		LexerData:  l.LexerData,
	}
}

func NewLexer(source SourceCode) *Lexer {
	return &Lexer{
		SourceCode: source,
		LexerData: LexerData{
			Row: 1,
		},
	}
}

func NewLexerFromFile(file *os.File) (*Lexer, error) {
	text, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &Lexer{
		SourceCode: SourceCode{
			FileName: file.Name(),
			Raw:      []rune(string(text)),
		},
		LexerData: LexerData{
			Row: 1,
		},
	}, nil
}

func (t *Lexer) Eof() bool {
	return t.Cur == len(t.Raw)
}

func (t *Lexer) currentRune() rune {
	if t.Cur < len(t.SourceCode.Raw) {
		return t.SourceCode.Raw[t.Cur]
	}
	return '\x00'
}

func (t Lexer) peekRune() (rune, error) {
	if t.Cur+1 >= len(t.SourceCode.Raw) {
		return 0, io.EOF
	}
	t.Cur += 1
	return t.SourceCode.Raw[t.Cur], nil
}

func (t *Lexer) eat() rune {
	current := t.SourceCode.Raw[t.Cur]
	t.Cur++
	return current
}

func (t *Lexer) consumeLeadingTrivia() string {
	start := t.Cur

TriviaLoop:
	for !t.Eof() {
		switch t.currentRune() {
		case '/':
			{
				if p, err := t.peekRune(); err != io.EOF && p == '*' {
					t.eat() // eat the /
					t.eat() // eat the *
				MultilineCommentLoop:
					for !t.Eof() {
						if p, err := t.peekRune(); err != io.EOF && t.currentRune() == '*' && p == '/' {
							t.eat()
							t.eat()
							break MultilineCommentLoop
						}
						if t.eat() == '\n' {
							t.Bol = t.Cur
							t.Row += 1
						}
					}
				} else {
					break TriviaLoop
				}
			}
		case '\n':
			{
				t.eat()
				t.Bol = t.Cur
				t.Row += 1
			}
		case ' ':
			t.eat()
		case '\t':
			t.eat()
		case '-':
			{
				if p, err := t.peekRune(); err != io.EOF && p == '-' {
					for !t.Eof() && t.currentRune() != '\n' {
						t.eat()
					}
				} else {
					break TriviaLoop
				}
			}
		default:
			break TriviaLoop
		}
	}
	end := t.Cur

	return string(t.SourceCode.Raw[start:end])
}

func (t *Lexer) consumeTrailingTrivia() string {
	start := t.Cur

TriviaLoop:
	for !t.Eof() {
		switch t.currentRune() {
		case '/':
			{
				if p, err := t.peekRune(); err != io.EOF && p == '*' {
					t.eat() // eat the /
					t.eat() // eat the *
				MultilineCommentLoop:
					for !t.Eof() {
						if p, err := t.peekRune(); err != io.EOF && t.currentRune() == '*' && p == '/' {
							t.eat()
							t.eat()
							break MultilineCommentLoop
						}
						if t.eat() == '\n' {
							t.Bol = t.Cur
							t.Row += 1
						}
					}
				} else {
					break TriviaLoop
				}
			}
		case '\n':
			{
				t.eat()
				t.Bol = t.Cur
				t.Row += 1
				break TriviaLoop
			}
		case ' ':
			t.eat()
		case '\t':
			t.eat()
		case '-':
			{
				if p, err := t.peekRune(); err != io.EOF && p == '-' {
					for !t.Eof() && t.currentRune() != '\n' {
						t.eat()
					}
				} else {
					break TriviaLoop
				}
			}
		default:
			break TriviaLoop
		}
	}
	end := t.Cur

	return string(t.SourceCode.Raw[start:end])
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func (t *Lexer) identifier() string {
	start := t.Cur
	for !t.Eof() && (unicode.IsLetter(t.currentRune()) || unicode.IsDigit(t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}
	end := t.Cur
	return string(t.Raw[start:end])
}

func (t *Lexer) decimalNumeric() (string, bool) {
	start := t.Cur

	hasPeriod := false
	hasExpon := false

	// eat as many digits as we can
	for !t.Eof() {
		if !unicode.IsDigit(t.currentRune()) {
			break
		}
		t.eat()
	}

	// check if we have a period
	if !hasPeriod && t.currentRune() == '.' {
		t.eat()
		hasPeriod = true
	}

	// eat as many digits as we can
	for !t.Eof() {
		if !unicode.IsDigit(t.currentRune()) {
			break
		}
		t.eat()
	}

	// check if we have a period
	if !hasExpon && (t.currentRune() == 'e' || t.currentRune() == 'E') {
		t.eat()
		hasExpon = true
	}

	// check for signs
	if t.currentRune() == '+' || t.currentRune() == '-' {
		t.eat()
	}

	// eat as many digits as we can
	for !t.Eof() {
		if !unicode.IsDigit(t.currentRune()) {
			break
		}
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end]), hasPeriod || hasExpon
}

func (t *Lexer) hexNumeric() string {
	start := t.Cur

	t.eat() // 0
	t.eat() // x

	for !t.Eof() && (unicode.Is(unicode.ASCII_Hex_Digit, t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

func (t *Lexer) binaryNumeric() string {
	start := t.Cur

	t.eat() // 0
	t.eat() // b

	for !t.Eof() && (t.currentRune() == '0' || t.currentRune() == '1' || t.currentRune() == '_') {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

var ASCII_Octal_Digit = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 0x0030, Hi: 0x0037, Stride: 1}, // '0'-'7'
	},
	LatinOffset: 1,
}

func (t *Lexer) octalNumeric() string {
	start := t.Cur

	t.eat() // 0
	t.eat() // 0

	for !t.Eof() && (unicode.Is(ASCII_Octal_Digit, t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

func (t Lexer) PeekToken() (tok token.Token) {
	tok = t.NextToken()
	return tok
}

func (t *Lexer) NextToken() (tok token.Token) {
	tok.FileLoc.FileName = t.SourceCode.FileName
	tok.SourceCode = t.SourceCode

	tok.LeadingTrivia = t.consumeLeadingTrivia()
	defer func() {
		tok.SourceRange.End = t.Cur
		tok.FileLoc.Line = t.LexerData.Row
		tok.FileLoc.Col = tok.SourceRange.Start - t.LexerData.Bol + 1
		tok.TrailingTrivia = t.consumeTrailingTrivia()
	}()

	if t.Eof() {
		tok.Kind = token.TokenKind_EOF
		tok.FileLoc.Line = t.LexerData.Row
		tok.FileLoc.Col = t.Cur
		return tok
	}

	tok.SourceRange.Start = t.Cur
	switch t.currentRune() {
	case ';', ',', '(', ')', '=', '+', '-', '*', '/':
		{
			r := t.currentRune()
			t.eat()
			tok.Kind = (token.TokenKind)(r)
			tok.Text = string(r)
			return tok
		}
	case '!':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				tok.Kind = token.TokenKind_neq
				tok.Text = "!="
				return tok
			}
			tok.Kind = '!'
			tok.Text = "!"
			return tok
		}
	case '>':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				tok.Kind = token.TokenKind_gte
				tok.Text = ">="
				return tok
			}
			tok.Kind = token.TokenKind_gt
			tok.Text = ">"
			return tok
		}
	case '<':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				tok.Kind = token.TokenKind_lte
				tok.Text = "<="
			}
			tok.Kind = token.TokenKind_lt
			tok.Text = "<"
			return tok
		}
	case '"':
		{
			// eat the first "
			t.eat()
			start := t.Cur
			prev := rune(0)
			for !t.Eof() {
				if t.currentRune() == '"' && prev != '\\' {
					break
				}
				prev = t.eat()
			}
			end := t.Cur
			// eat the last "
			t.eat()
			tok.Kind = token.TokenKind_Identifier
			tok.Text = string(t.Raw[start:end])
			tok.OpenQuote = '"'
			tok.CloseQuote = '"'
			return tok
		}
	case '[':
		{
			// eat the first [
			t.eat()
			start := t.Cur
			for !t.Eof() {
				if t.currentRune() == ']' {
					break
				}
				t.eat()
			}
			end := t.Cur
			// eat the last ]
			t.eat()
			tok.Kind = token.TokenKind_Identifier
			tok.Text = string(t.Raw[start:end])
			tok.OpenQuote = '['
			tok.CloseQuote = ']'
			return tok
		}
	case '`':
		{
			// eat the first `
			t.eat()
			start := t.Cur
			prev := rune(0)
			for !t.Eof() {
				if t.currentRune() == '`' && prev != '\\' {
					break
				}
				prev = t.eat()
			}
			end := t.Cur
			// eat the last `
			t.eat()
			tok.Kind = token.TokenKind_Identifier
			tok.Text = string(t.Raw[start:end])
			tok.OpenQuote = '`'
			tok.CloseQuote = '`'
			return tok
		}
	case '\'':
		{
			// eat the first '
			t.eat()
			tok.Kind = token.TokenKind_StringLiteral
			start := t.Cur
			prev := rune(0)
			for !t.Eof() {
				if t.currentRune() == '\'' && prev != '\\' {
					break
				}
				prev = t.eat()
			}
			// eat the last '
			t.eat()
			end := t.Cur
			tok.Text = string(t.Raw[start:end])
			tok.OpenQuote = '\''
			tok.CloseQuote = '\''
			return tok
		}
	case '.':
		{
			if p, err := t.peekRune(); err != io.EOF && unicode.IsDigit(p) {
				text, isFloat := t.decimalNumeric()
				if isFloat {
					tok.Kind = token.TokenKind_FloatNumericLiteral
				} else {
					tok.Kind = token.TokenKind_IntegerNumericLiteral
				}
				tok.Text = text
				return tok
			}
			t.eat()
			tok.Kind = token.TokenKind_Period
			tok.Text = "."
			return tok
		}
	case '0':
		{
			switch p, err := t.peekRune(); err != io.EOF {
			case unicode.ToLower(p) == 'x':
				{
					tok.Kind = token.TokenKind_HexNumericLiteral
					tok.Text = t.hexNumeric()
					return tok
				}
			case unicode.ToLower(p) == 'b':
				{
					tok.Kind = token.TokenKind_BinaryNumericLiteral
					tok.Text = t.binaryNumeric()
					return tok
				}
			case p == '0':
				{
					tok.Kind = token.TokenKind_OctalNumericLiteral
					tok.Text = t.octalNumeric()
					return tok
				}
			}
		}
		fallthrough
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		{
			text, isFloat := t.decimalNumeric()
			tok.Text = text
			if isFloat {
				tok.Kind = token.TokenKind_FloatNumericLiteral
			} else {
				tok.Kind = token.TokenKind_IntegerNumericLiteral
			}
			return tok
		}
	}

	if isIdentifierStart(t.currentRune()) {
		tok.Kind = token.TokenKind_Identifier
		tok.Text = t.identifier()
		if kind, ok := token.KeywordIndex.GetValue(strings.ToLower(tok.Text)); ok {
			tok.Kind = kind
		}
		return tok
	}

	return tok
}
