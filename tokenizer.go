package main

import (
	"io"
	"os"
	"strings"
	"unicode"
)

type SourceCode struct {
	FileName string
	Raw      []rune
}

type TokenizerData struct {
	Cur int
	Bol int
	Row int
}

type Tokenizer struct {
	SourceCode
	TokenizerData
	CurrentToken Token
}

func NewTokenizerFromFile(file *os.File) (*Tokenizer, error) {
	text, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &Tokenizer{
		SourceCode: SourceCode{
			FileName: file.Name(),
			Raw:      []rune(string(text)),
		},
		TokenizerData: TokenizerData{
			Cur: 0,
			Row: 1,
		},
	}, nil
}

func (t *Tokenizer) Eof() bool {
	return t.Cur == len(t.Raw)
}

func (t *Tokenizer) currentRune() rune {
	return t.SourceCode.Raw[t.Cur]
}

func (t Tokenizer) peekRune() (rune, error) {
	if t.Cur+1 >= len(t.SourceCode.Raw) {
		return 0, io.EOF
	}
	t.Cur += 1
	return t.SourceCode.Raw[t.Cur], nil
}

func (t *Tokenizer) eat() rune {
	current := t.SourceCode.Raw[t.Cur]
	t.Cur++
	return current
}

func (t *Tokenizer) consumeLeadingTrivia() string {
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
			{
				t.eat()
			}
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

func (t *Tokenizer) consumeTrailingTrivia() string {
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
			{
				t.eat()
			}
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

func (t *Tokenizer) identifier() string {
	start := t.Cur
	for !t.Eof() && (unicode.IsLetter(t.currentRune()) || unicode.IsDigit(t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}
	end := t.Cur
	return string(t.Raw[start:end])
}

func (t *Tokenizer) decimalNumeric() string {
	start := t.Cur
	hasPeriod := false
	hasExpon := false

	for !t.Eof() {

		if !hasPeriod && t.currentRune() == '.' {
			t.eat()
			hasPeriod = true
			continue
		}

		if !hasExpon && t.currentRune() == 'e' {
			t.eat()
			hasExpon = true
			continue
		}

		if !unicode.IsDigit(t.currentRune()) {
			break
		}

		t.eat()
	}

	if t.currentRune() == 'f' {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

func (t *Tokenizer) hexNumeric() string {
	start := t.Cur

	t.eat() // 0
	t.eat() // x

	for !t.Eof() && (unicode.Is(unicode.ASCII_Hex_Digit, t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

func (t *Tokenizer) binaryNumeric() string {
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

func (t *Tokenizer) octalNumeric() string {
	start := t.Cur

	t.eat() // 0
	t.eat() // 0

	for !t.Eof() && (unicode.Is(ASCII_Octal_Digit, t.currentRune()) || t.currentRune() == '_') {
		t.eat()
	}

	end := t.Cur
	return string(t.Raw[start:end])
}

func (t Tokenizer) PeekToken() (token Token) {
	token = t.NextToken()
	return token
}

func (t *Tokenizer) NextToken() (token Token) {
	token.FileLoc.FileName = t.SourceCode.FileName
	if t.Eof() {
		token.Kind = TokenKind_EOF
		token.FileLoc.Line = t.TokenizerData.Row
		token.FileLoc.Col = t.Cur
		return token
	}

	token.LeadingTrivia = t.consumeLeadingTrivia()
	defer func() {
		token.SourceRange.End = t.Cur
		token.FileLoc.Line = t.TokenizerData.Row
		token.FileLoc.Col = token.SourceRange.Start - t.TokenizerData.Bol + 1
		token.TrailingTrivia = t.consumeTrailingTrivia()
	}()

	token.SourceRange.Start = t.Cur
	switch t.currentRune() {
	case ';', ',', '(', ')', '=', '-', '*', '/':
		{
			r := t.currentRune()
			t.eat()
			token.Kind = (TokenKind)(r)
			token.Text = string(r)
			return token
		}
	case '!':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				token.Kind = TokenKind_neq
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
				token.Kind = TokenKind_gte
				token.Text = ">="
				return token
			}
			token.Kind = TokenKind_gt
			token.Text = ">"
			return token
		}
	case '<':
		{
			t.eat()
			if t.currentRune() == '=' {
				t.eat()
				token.Kind = TokenKind_lte
				token.Text = "<="
			}
			token.Kind = TokenKind_lt
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
			// eat the last "
			t.eat()
			end := t.Cur
			token.Kind = TokenKind_Identifier
			token.Text = string(t.Raw[start:end])
			return token
		}
	case '\'':
		{
			// eat the first '
			t.eat()
			token.Kind = TokenKind_StringLiteral
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
				token.Kind = TokenKind_DecimalNumericLiteral
				token.Text = t.decimalNumeric()
				return token
			}
			t.eat()
			token.Kind = TokenKind_Period
			token.Text = "."
			return token
		}
	case '0':
		{
			switch p, err := t.peekRune(); err != io.EOF {
			case unicode.ToLower(p) == 'x':
				{
					token.Kind = TokenKind_HexNumericLiteral
					token.Text = t.hexNumeric()
					return token
				}
			case unicode.ToLower(p) == 'b':
				{
					token.Kind = TokenKind_BinaryNumericLiteral
					token.Text = t.binaryNumeric()
					return token
				}
			case p == '0':
				{
					token.Kind = TokenKind_OctalNumericLiteral
					token.Text = t.octalNumeric()
					return token
				}
			}
		}
		fallthrough
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		{
			token.Kind = TokenKind_DecimalNumericLiteral
			token.Text = t.decimalNumeric()
			return token
		}
	}

	if isIdentifierStart(t.currentRune()) {
		token.Kind = TokenKind_Identifier
		token.Text = t.identifier()
		if kind, ok := keywordIndex.GetValue(strings.ToLower(token.Text)); ok {
			token.Kind = kind
		}
		return token
	}

	return token
}
