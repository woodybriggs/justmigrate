package main

import (
	"io"
	"os"
	"strings"
	"unicode"
)

type Location struct {
	FileName string
	Line     int
	Col      int
}

type Lexer struct {
	FileName      string
	Text          []rune
	CurrentPos    int
	PreviousToken TokenKind
}

func NewLexer(text string) *Lexer {
	return &Lexer{
		Text:          []rune(text),
		CurrentPos:    0,
		PreviousToken: 0,
	}
}

func NewLexerFromFile(file *os.File) (*Lexer, error) {
	text, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	lex := NewLexer(string(text))
	lex.FileName = file.Name()

	return lex, nil
}

func (l Lexer) current() rune {
	return l.Text[l.CurrentPos]
}

func (l Lexer) peek() (rune, error) {
	l.CurrentPos += 1
	if l.Eof() {
		return '\uFFFF', io.EOF
	}
	return l.current(), nil
}

func (l *Lexer) consume() {
	l.CurrentPos += 1
}

func (l *Lexer) backup() {
	l.CurrentPos -= 1
}

type ComparableFn func(l Lexer) bool

func (l *Lexer) consumeWhile(fn ComparableFn) {
	for !l.Eof() && fn(*l) {
		l.consume()
	}
}

func (l Lexer) Eof() bool {
	return l.CurrentPos == len(l.Text)
}

func (l Lexer) tokenFromCurrent(kind TokenKind) Token {
	tok := Token{
		Kind: kind,
		Text: string(l.current()),
	}
	return tok
}

func (l Lexer) tokenFromRange(kind TokenKind, start int, end int) Token {
	tok := Token{
		Kind: kind,
		Text: string(l.Text[start:end]),
	}
	return tok
}

func (l *Lexer) SkipWhitespace() {
	for !l.Eof() && unicode.IsSpace(l.current()) {
		l.consume()
	}
}

func isMultilineCommentStart(c rune, p rune) bool {
	return c == '/' && p == '*'
}

func isMultilineCommentEnd(c rune, p rune) bool {
	return c == '*' && p == '/'
}

func (l Lexer) PeekToken(count int) Token {
	res := Token{}
	for range count {
		res = l.ConsumeToken()
	}
	return res
}

func (l *Lexer) consumeLeadingTrivia() string {
	start := l.CurrentPos

LeadingTriviaLoop:
	for !l.Eof() {
		c := l.current()
		if l.PreviousToken != 0 && c == '\n' {
			break
		}
		if unicode.IsSpace(c) {
			l.consume()
			continue
		}
		p, err := l.peek()
		if err != nil {
			break
		}
		if l.PreviousToken == 0 && c == '-' && p == '-' {
			l.consumeWhile(func(lex Lexer) bool { return lex.current() != '\n' })
			continue
		}
		if isMultilineCommentStart(c, p) {
		MultiLineComment:
			for !l.Eof() {
				c := l.current()
				p, err := l.peek()
				if err != nil {
					break LeadingTriviaLoop
				}
				if isMultilineCommentEnd(c, p) {
					l.consume()
					l.consume()
					break MultiLineComment
				} else {
					l.consume()
				}
			}
			continue
		}
		break LeadingTriviaLoop
	}
	end := l.CurrentPos
	return string(l.Text[start:end])
}

func (l *Lexer) consumeTrailingTrivia() string {
	start := l.CurrentPos
LeadingTriviaLoop:
	for !l.Eof() {
		c := l.current()
		if c == '\n' {
			l.consume()
			break
		}
		if unicode.IsSpace(c) {
			l.consume()
			continue
		}
		if c == '-' {
			p, _ := l.peek()
			if p == '-' {
				l.consume()
				l.consume()
				l.consumeWhile(func(lex Lexer) bool { return lex.current() != '\n' })
				l.consume()
				break
			}
		}
		if c == '/' {
			p, _ := l.peek()
			if p == '*' {
				l.consume()
				l.consume()
				for !l.Eof() {
					c2 := l.current()
					p2, _ := l.peek()
					if c2 == '*' && p2 == '/' {
						l.consume()
						l.consume()
						break
					}
					l.consume()
				}
				l.consume()
				continue
			}
			continue
		}
		break LeadingTriviaLoop
	}
	end := l.CurrentPos
	return string(l.Text[start:end])
}

func (l *Lexer) ConsumeToken() Token {
	tok := Token{}
	if l.Eof() {
		l.PreviousToken = tok.Kind
		return tok
	}

	leadingTrivia := l.consumeLeadingTrivia()

	start := l.CurrentPos
	current := l.current()
	switch current {
	case '0':
		peeked, _ := l.peek()
		if unicode.ToLower(peeked) == 'x' {
			tok = l.HexNumeric()
			l.PreviousToken = tok.Kind
			tok.LeadingTrivia = leadingTrivia
			tok.TrailingTrivia = l.consumeTrailingTrivia()
			return tok
		}
		fallthrough
	case '1', '2', '3', '4', '5', '6', '7', '8', '9':
		tok = l.Numeric()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '.':
		peeked, _ := l.peek()
		if unicode.IsDigit(peeked) {
			tok = l.Numeric()
			l.PreviousToken = tok.Kind
			tok.LeadingTrivia = leadingTrivia
			tok.TrailingTrivia = l.consumeTrailingTrivia()
			return tok
		}
		l.consume()
		tok = l.tokenFromCurrent(TokenKind_Period)
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '-':
		l.consume()
		tok = l.tokenFromCurrent(TokenKind_Minus)
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '`':
		l.consume()
		start = l.CurrentPos
		l.consumeWhile(func(lex Lexer) bool { return lex.current() != '`' })
		end := l.CurrentPos
		l.consume()
		tok = l.tokenFromRange(TokenKind_Identifier, start, end)
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '+':
		tok = l.tokenFromCurrent(TokenKind_Plus)
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '(':
		tok = l.tokenFromCurrent(TokenKind_LParen)
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case ')':
		tok = l.tokenFromCurrent(TokenKind_RParen)
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case ',':
		tok = l.tokenFromCurrent(TokenKind_Comma)
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case ';':
		tok = l.tokenFromCurrent(TokenKind_SemiColon)
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '=':
		tok = l.tokenFromCurrent('=')
		l.consume()
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '\'':
		l.consume()
		start = l.CurrentPos
		l.consumeWhile(func(lex Lexer) bool { return lex.current() != '\'' })
		end := l.CurrentPos
		l.consume()
		tok = l.tokenFromRange(TokenKind_StringLiteral, start, end)
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	case '"':
		l.consume()
		start = l.CurrentPos
		l.consumeWhile(func(lex Lexer) bool { return lex.current() != '"' })
		end := l.CurrentPos
		l.consume()
		tok = l.tokenFromRange(TokenKind_Identifier, start, end)
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	}

	for !l.Eof() && (unicode.IsLetter(l.current()) || l.current() == '_') {
		l.consume()
	}
	end := l.CurrentPos

	tok = l.tokenFromRange(TokenKind_Identifier, start, end)

	keywordmatch := strings.ToLower(tok.Text)
	if kind, ok := keywordIndex.GetValue(keywordmatch); ok {
		tok.Kind = kind
		l.PreviousToken = tok.Kind
		tok.LeadingTrivia = leadingTrivia
		tok.TrailingTrivia = l.consumeTrailingTrivia()
		return tok
	}
	tok.LeadingTrivia = leadingTrivia
	tok.TrailingTrivia = l.consumeTrailingTrivia()
	l.PreviousToken = tok.Kind
	return tok
}

func (l *Lexer) HexNumeric() Token {
	current := l.current()
	peeked, err := l.peek()
	if err != nil {
		return Token{}
	}

	if current != '0' || unicode.ToLower(peeked) != 'x' {
		return Token{}
	}

	start := l.CurrentPos
	l.consume()
	l.consume()

	for !l.Eof() && unicode.Is(unicode.ASCII_Hex_Digit, l.current()) {
		l.consume()
	}
	end := l.CurrentPos

	return l.tokenFromRange(TokenKind_HexNumericLiteral, start, end)
}

func (l *Lexer) Numeric() Token {
	start := l.CurrentPos
	for !l.Eof() && unicode.IsDigit(l.current()) {
		l.consume()
	}
	if l.current() == '.' {
		l.consume()
		for !l.Eof() && unicode.IsDigit(l.current()) {
			l.consume()
		}
	}
	end := l.CurrentPos
	return l.tokenFromRange(TokenKind_DecimalNumericLiteral, start, end)
}
