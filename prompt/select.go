package prompt

import "errors"

var ErrInvalidSelection = errors.New("Invalid Selection")

type SelectOption struct {
	Label string
	Value any
}

type Select struct {
	hovered      int
	selected     int
	optionsCount int
	shouldExit   bool
}

func (s *Select) Do(terminal *Terminal, title string, options []SelectOption) (choice int, err error) {
	s.optionsCount = len(options)
	s.shouldExit = false
	s.hovered = 0
	s.selected = -1

	for !s.shouldExit {

		// clear screen
		terminal.Sequence(CUP, nil)
		terminal.Sequence(ED, []int{0})

		// render title
		terminal.LineOfText(title)

		// render description
		terminal.ItalicStyle()
		terminal.LineOfText("use arrow keys to navigate, space to select and enter to confirm selection")
		terminal.EndStyles()

		// render options
		for i, option := range options {
			if i == s.hovered {
				terminal.Rune(0x25BA)
			} else {
				terminal.Rune(' ')
			}

			if i == s.selected {
				terminal.Rune(0x25CF)
				terminal.Rune(' ')
				terminal.BoldStyle()
				terminal.LineOfText(option.Label)
				terminal.EndStyles()
			} else {
				terminal.Rune(0x25CB)
				terminal.Rune(' ')
				terminal.LineOfText(option.Label)
			}
		}

		// write the rendering
		err = terminal.Flush()
		if err != nil {
			s.shouldExit = true
		}

		// wait for next input to read
		err = terminal.NextEvent(s)
		if err != nil {
			s.shouldExit = true
		}
	}
	return s.selected, err
}

func (s *Select) OnKey(c rune) error {
	switch c {
	case ' ':
		if s.selected == s.hovered {
			s.selected = -1
		} else {
			s.selected = s.hovered
		}
	}
	return nil
}

func (s *Select) OnControl(ctl ANSIIControlCharacter) error {
	switch ctl {
	case EXT:
		s.shouldExit = true
		return ErrInvalidSelection
	case LF, CR:
		if s.selected > -1 {
			s.shouldExit = true
		}
	}
	return nil
}

func (s *Select) OnCSI(cmd ANSIISequenceCommand, params []int) error {

	if len(params) == 0 {
		params = append(params, 1)
	} else if params[0] == 0 {
		params[0] = 1
	}

	switch cmd {
	case ANSIISequenceCommand(EXT):
		s.shouldExit = true
	case CUD:
		s.hovered = (s.hovered + int(params[0])) % s.optionsCount
	case CUU:
		s.hovered = (s.hovered - int(params[0]) + s.optionsCount) % s.optionsCount
	}
	return nil
}
