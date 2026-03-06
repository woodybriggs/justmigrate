package prompt

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

type EscapeSequenceTestCase struct {
	Sequence string
	Result
}

type Result struct {
	Cmd    ANSIISequenceCommand
	Params []int
}

type handler struct {
	Results []Result
}

func (handler *handler) OnCSI(cmd ANSIISequenceCommand, params []int) error {
	handler.Results = append(handler.Results, Result{Cmd: cmd, Params: params})
	return nil
}

func (handler *handler) OnControl(ctl ANSIIControlCharacter) error {
	handler.Results = append(handler.Results, Result{Cmd: ANSIISequenceCommand(EXT), Params: nil})
	return nil
}

func (handler *handler) OnKey(r rune) error {
	return nil
}

func TestANSIIEscapeSequenceTokenizer(t *testing.T) {

	testCases := []EscapeSequenceTestCase{
		{
			Sequence: "\x1b[1A",
			Result: Result{
				Cmd:    'A',
				Params: []int{1},
			},
		},
		{
			Sequence: "\x1b[2B",
			Result: Result{
				Cmd:    'B',
				Params: []int{2},
			},
		},
		{
			Sequence: "\x1b[3C",
			Result: Result{
				Cmd:    'C',
				Params: []int{3},
			},
		},
		{
			Sequence: "\x1b[4D",
			Result: Result{
				Cmd:    'D',
				Params: []int{4},
			},
		},
		{
			Sequence: "\x03",
			Result: Result{
				Cmd:    '\x03',
				Params: nil,
			},
		},
	}

	buf := &bytes.Buffer{}
	reader := bufio.NewReader(buf)
	tokenizer := ANSIIEscapeSequenceTokenizer{Reader: reader}

	handler := handler{}

	for i, test := range testCases {
		buf.WriteString(test.Sequence)
		err := tokenizer.NextEvent(&handler)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

		expected := test.Result
		got := handler.Results[i]

		if expected.Cmd != got.Cmd {
			fmt.Printf("expected '%c' got '%c'", expected.Cmd, got.Cmd)
			t.Fail()
		}

		if expected.Params == nil && got.Params == nil {
			continue
		}

		if len(expected.Params) != len(got.Params) {
			fmt.Printf("expected params length to be the same: expected '%d' got '%d'", len(expected.Params), len(got.Params))
			t.Fail()
		}

		for i := range len(got.Params) {
			expP, gotP := expected.Params[i], got.Params[i]

			if expP != gotP {
				fmt.Printf("params mismatched: expected '%d' got '%d'", expP, gotP)
				t.Fail()
			}
		}
	}
}
