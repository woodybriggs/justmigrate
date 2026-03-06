package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/term"
)

var ErrFailedToStart = errors.New("failed to start terminal interface")

type TerminalHandler interface {
	OnKey(c rune) error
	OnCSI(cmd ANSIISequenceCommand, params []int) error
	OnControl(ctl ANSIIControlCharacter) error
}

type Terminal struct {
	prevState *term.State
	tokenizer ANSIIEscapeSequenceTokenizer
	writer    *bufio.Writer
}

func (t *Terminal) Sequence(cmd ANSIISequenceCommand, params []int) {
	fmt.Fprintf(t.writer, "%c[", byte(ESC))
	for i, param := range params {
		fmt.Fprintf(t.writer, "%s", strconv.FormatInt(int64(param), 10))
		if i < len(params)-1 {
			fmt.Fprint(t.writer, ";")
		}
	}
	fmt.Fprintf(t.writer, "%c", byte(cmd))
}

func (t *Terminal) ItalicStyle() {
	t.Sequence('m', []int{3})
}

func (t *Terminal) BoldStyle() {
	t.Sequence('m', []int{1})
}

func (t *Terminal) EndStyles() {
	t.Sequence('m', []int{0})
}

func (t *Terminal) Control(ctl ANSIIControlCharacter) {
	fmt.Fprintf(t.writer, "%c", byte(ctl))
}

func (t *Terminal) Rune(r rune) {
	fmt.Fprintf(t.writer, "%c", r)
}

func (t *Terminal) LineOfText(l string) {
	fmt.Fprintf(t.writer, "%s", l)
	t.Control(CR)
	t.Control(LF)
}

func (t *Terminal) Flush() error {
	return t.writer.Flush()
}

func (t *Terminal) Start() error {

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("%w in raw mode: %w", ErrFailedToStart, err)
	}
	t.prevState = oldState

	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	t.tokenizer = ANSIIEscapeSequenceTokenizer{reader: reader}
	t.writer = writer

	return nil
}

func (t *Terminal) Restore() {
	term.Restore(int(os.Stdin.Fd()), t.prevState)
}

func (t *Terminal) NextEvent(handler TerminalHandler) error {
	if t.tokenizer.reader == nil {
		return ErrFailedToStart
	}
	return t.tokenizer.NextEvent(handler)
}

type ANSIIControlCharacter byte

const (
	NUL ANSIIControlCharacter = '\x00'
	EXT ANSIIControlCharacter = '\x03'
	BS  ANSIIControlCharacter = '\x08'
	HT  ANSIIControlCharacter = '\x09'
	LF  ANSIIControlCharacter = '\x0A'
	CR  ANSIIControlCharacter = '\x0D'
	ESC ANSIIControlCharacter = '\x1B'
	DEL ANSIIControlCharacter = '\x7F'
)

type ANSIISequenceCommand int

const (
	NOP ANSIISequenceCommand = -1
	// Cursor Home
	CUP ANSIISequenceCommand = 'H'
	// Cursor to Absolute Position
	HVP ANSIISequenceCommand = 'f'
	// Cursor Up
	CUU ANSIISequenceCommand = 'A'
	// Cursor Down
	CUD ANSIISequenceCommand = 'B'
	// Cursor Forward
	CUF ANSIISequenceCommand = 'C'
	// Cursor Backward
	CUB ANSIISequenceCommand = 'D'

	CNL ANSIISequenceCommand = 'E'
	CPL ANSIISequenceCommand = 'F'
	CHA ANSIISequenceCommand = 'G'

	// Erase Display
	ED ANSIISequenceCommand = 'J'
)

type ANSIIEscapeSequenceTokenizer struct {
	reader     *bufio.Reader
	paramBuf   [8]int
	paramCount int
}

func (t *ANSIIEscapeSequenceTokenizer) NextEvent(handler TerminalHandler) error {
	r, _, err := t.reader.ReadRune()
	if err != nil {
		return err
	}

	switch r {
	case rune(EXT):
		return handler.OnControl(EXT)
	case rune(LF):
		return handler.OnControl(LF)
	case rune(CR):
		return handler.OnControl(CR)
	case rune(ESC):
		cmd, params, err := t.ansiiEscapeSequence()
		if err != nil {
			return err
		}
		return handler.OnCSI(cmd, params)
	default:
		return handler.OnKey(r)
	}
}

func (t *ANSIIEscapeSequenceTokenizer) ansiiEscapeSequence() (ANSIISequenceCommand, []int, error) {
	n, err := t.reader.ReadByte()
	if err != nil {
		return NOP, nil, err
	}

	switch n {
	case '[':
		return t.csiSequence()
	default:
		return NOP, nil, nil
	}
}

func isDigit(b byte) bool {
	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	default:
		return false
	}
}

func (t *ANSIIEscapeSequenceTokenizer) csiSequence() (ANSIISequenceCommand, []int, error) {

	var n byte
	var err error
	t.paramCount = 0
	clear(t.paramBuf[0:8])

	for !errors.Is(err, io.EOF) {

		n, err = t.reader.ReadByte()

		if isDigit(n) {
			var number int = 0
			number, n, err = t.numberParameter(n)
			if err != nil {
				break
			}
			if t.paramCount < len(t.paramBuf) {
				t.paramBuf[t.paramCount] = number
				t.paramCount += 1
			}
		}

		switch ANSIISequenceCommand(n) {
		case ';':
			continue
		case CUU, CUD, CUF, CUB:
			return ANSIISequenceCommand(n), t.paramBuf[:t.paramCount], nil
		default:
			// swallow the sequence and let the ui continue
			return NOP, nil, nil
		}
	}

	return NOP, nil, err
}

func (t *ANSIIEscapeSequenceTokenizer) numberParameter(first byte) (int, byte, error) {
	str := []byte{first}
	var n byte
	var err error

	for {
		n, err = t.reader.ReadByte()
		if err != nil || !isDigit(n) {
			break
		}
		str = append(str, n)
	}

	value, parseErr := strconv.ParseInt(string(str), 10, 64)
	if parseErr != nil {
		return 0, n, parseErr
	}

	return int(value), n, err
}
