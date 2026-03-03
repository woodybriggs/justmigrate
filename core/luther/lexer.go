package luther

import (
	"io"
	"os"
	"strings"
	"unicode"
	"woodybriggs/justmigrate/core/tik"
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

func (t Lexer) PeekToken() (token tik.Token) {
	token = t.NextToken()
	return token
}

func (t *Lexer) NextToken() (token tik.Token) {
	token.FileLoc.FileName = t.SourceCode.FileName
	token.SourceCode = t.SourceCode

	token.LeadingTrivia = t.consumeLeadingTrivia()
	defer func() {
		token.SourceRange.End = t.Cur
		token.FileLoc.Line = t.LexerData.Row
		token.FileLoc.Col = token.SourceRange.Start - t.LexerData.Bol + 1
		token.TrailingTrivia = t.consumeTrailingTrivia()
	}()

	if t.Eof() {
		token.Kind = tik.TokenKind_EOF
		token.FileLoc.Line = t.LexerData.Row
		token.FileLoc.Col = t.Cur
		return token
	}

	token.SourceRange.Start = t.Cur
	switch t.currentRune() {
	case ';', ',', '(', ')', '=', '+', '-', '*', '/':
		{
			r := t.currentRune()
			t.eat()
			token.Kind = (tik.TokenKind)(r)
			token.Text = string(r)
			return token
		}
	case '!':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				token.Kind = tik.TokenKind_neq
				token.Text = "!="
				return token
			}
			token.Kind = '!'
			token.Text = "!"
			return token
		}
	case '>':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				token.Kind = tik.TokenKind_gte
				token.Text = ">="
				return token
			}
			token.Kind = tik.TokenKind_gt
			token.Text = ">"
			return token
		}
	case '<':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				token.Kind = tik.TokenKind_lte
				token.Text = "<="
			}
			token.Kind = tik.TokenKind_lt
			token.Text = "<"
			return token
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
			token.Kind = tik.TokenKind_Identifier
			token.Text = string(t.Raw[start:end])
			return token
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
			token.Kind = tik.TokenKind_Identifier
			token.Text = string(t.Raw[start:end])
			return token
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
			token.Kind = tik.TokenKind_Identifier
			token.Text = string(t.Raw[start:end])
			return token
		}
	case '\'':
		{
			// eat the first '
			t.eat()
			token.Kind = tik.TokenKind_StringLiteral
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
			token.Text = string(t.Raw[start:end])
			return token
		}
	case '.':
		{
			if p, err := t.peekRune(); err != io.EOF && unicode.IsDigit(p) {
				text, isFloat := t.decimalNumeric()
				if isFloat {
					token.Kind = tik.TokenKind_FloatNumericLiteral
				} else {
					token.Kind = tik.TokenKind_IntegerNumericLiteral
				}
				token.Text = text
				return token
			}
			t.eat()
			token.Kind = tik.TokenKind_Period
			token.Text = "."
			return token
		}
	case '0':
		{
			switch p, err := t.peekRune(); err != io.EOF {
			case unicode.ToLower(p) == 'x':
				{
					token.Kind = tik.TokenKind_HexNumericLiteral
					token.Text = t.hexNumeric()
					return token
				}
			case unicode.ToLower(p) == 'b':
				{
					token.Kind = tik.TokenKind_BinaryNumericLiteral
					token.Text = t.binaryNumeric()
					return token
				}
			case p == '0':
				{
					token.Kind = tik.TokenKind_OctalNumericLiteral
					token.Text = t.octalNumeric()
					return token
				}
			}
		}
		fallthrough
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		{
			text, isFloat := t.decimalNumeric()
			token.Text = text
			if isFloat {
				token.Kind = tik.TokenKind_FloatNumericLiteral
			} else {
				token.Kind = tik.TokenKind_IntegerNumericLiteral
			}
			return token
		}
	}

	if isIdentifierStart(t.currentRune()) {
		token.Kind = tik.TokenKind_Identifier
		token.Text = t.identifier()
		if kind, ok := tik.KeywordIndex.GetValue(strings.ToLower(token.Text)); ok {
			token.Kind = kind
		}
		return token
	}

	return token
}
