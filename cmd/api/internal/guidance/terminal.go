package guidance

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

type terminalForm struct {
	in        *bufio.Reader
	out       *os.File
	fd        int
	summary   func() string
	notice    string
	pending   strings.Builder
	width     int
	height    int
	cursorRow int
	page      int
}

func newTerminalForm() *terminalForm {
	var form terminalForm
	form.in = bufio.NewReader(os.Stdin)
	form.out = os.Stderr
	form.fd = int(os.Stdin.Fd())
	return &form
}

func (f *terminalForm) Write(data []byte) (int, error) {
	return f.pending.Write(data)
}

func (f *terminalForm) content() string {
	var content strings.Builder
	content.WriteString(f.notice)
	content.WriteString(f.pending.String())
	if f.summary != nil {
		content.WriteString(f.summary())
	}
	return strings.TrimRight(content.String(), "\n")
}

func (f *terminalForm) finish() (err error) {
	defer func() {
		_, restoreErr := fmt.Fprint(f.out, "\x1b[?25h")
		err = errors.Join(err, restoreErr)
	}()
	if f.cursorRow == 0 {
		return nil
	}
	if err := f.size(); err != nil {
		return err
	}
	f.pending.Reset()
	f.notice = ""
	if err := f.draw(nil, -1); err != nil {
		return err
	}
	f.cursorRow = 0
	f.page = 0
	return nil
}

func (f *terminalForm) String(label string, defaults ...string) (string, error) {
	if err := f.size(); err != nil {
		return "", err
	}
	text := label
	if len(defaults) > 0 && defaults[0] != "" {
		value, rest := splitColumns(defaults[0], max(1, f.width-uniseg.StringWidth(label)-8))
		if rest != "" {
			value += "…"
		}
		text += " [" + value + "]"
	}
	value, err := f.text(text+": ", false)
	value = strings.TrimSpace(value)
	if value == "" && len(defaults) > 0 {
		value = defaults[0]
	}
	return value, err
}

func (f *terminalForm) Secret(label string) (string, error) {
	return f.text(label+" (hidden): ", true)
}

func (f *terminalForm) text(label string, hidden bool) (value string, err error) {
	state, err := term.MakeRaw(f.fd)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, term.Restore(f.fd, state)) }()
	defer f.pending.Reset()
	f.page = 0
	var input []string
	cursor := 0
	for {
		if err := f.size(); err != nil {
			return "", err
		}
		line, column := label, uniseg.StringWidth(label)
		if !hidden {
			line += strings.ReplaceAll(strings.Join(input, ""), "\t", "⇥")
			column += uniseg.StringWidth(strings.ReplaceAll(strings.Join(input[:cursor], ""), "\t", "⇥"))
		}
		if err := f.draw([]string{line}, column); err != nil {
			return "", err
		}
		key, err := f.key()
		if err != nil {
			return "", err
		}
		switch key {
		case "\r", "\n":
			return strings.Join(input, ""), nil
		case "\x7f", "\b":
			if cursor > 0 {
				input = slices.Delete(input, cursor-1, cursor)
				cursor--
			}
		case "\x1b[D", "\x02":
			cursor = max(0, cursor-1)
		case "\x1b[C", "\x06":
			cursor = min(len(input), cursor+1)
		case "\x1b[H", "\x1b[1~", "\x01":
			cursor = 0
		case "\x1b[F", "\x1b[4~", "\x05":
			cursor = len(input)
		case "\x1b[3~":
			if cursor < len(input) {
				input = slices.Delete(input, cursor, cursor+1)
			}
		case "\x15":
			input = input[cursor:]
			cursor = 0
		case "\x0b":
			input = input[:cursor]
		case "\x1b[5~":
			f.page = max(0, f.page-1)
		case "\x1b[6~":
			f.page++
		default:
			if key != "\t" && strings.ContainsFunc(key, unicode.IsControl) {
				continue
			}
			prefix := strings.Join(input[:cursor], "") + key
			graphemes := uniseg.NewGraphemes(prefix)
			var updated []string
			for graphemes.Next() {
				updated = append(updated, graphemes.Str())
			}
			cursor, input = len(updated), append(updated, input[cursor:]...)
		}
	}
}

func (f *terminalForm) SelectOrdered(label string, options []prompt.SelectOption, defaults ...string) (value string, err error) {
	state, err := term.MakeRaw(f.fd)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, term.Restore(f.fd, state)) }()
	defer f.pending.Reset()
	f.page = 0
	selected := 0
	if len(defaults) > 0 {
		for i, option := range options {
			if option.Key == defaults[0] {
				selected = i
				break
			}
		}
	}
	for {
		if err := f.size(); err != nil {
			return "", err
		}
		visible := min(len(options), max(1, f.height/2-3))
		top := min(max(0, selected-visible+1), len(options)-visible)
		lines := []string{label}
		if visible < len(options) {
			lines = append(lines, fmt.Sprintf("  %d–%d of %d · ↑/↓ to move", top+1, top+visible, len(options)))
		}
		for i := top; i < top+visible; i++ {
			prefix := "    "
			if i == selected {
				prefix = "  ❯ "
			}
			lines = append(lines, prefix+options[i].Label)
		}
		if err := f.draw(lines, -1); err != nil {
			return "", err
		}
		key, err := f.key()
		if err != nil {
			return "", err
		}
		switch key {
		case "\x1b[A":
			selected = (selected + len(options) - 1) % len(options)
		case "\x1b[B":
			selected = (selected + 1) % len(options)
		case "\x1b[5~":
			f.page = max(0, f.page-1)
		case "\x1b[6~":
			f.page++
		case "\r", "\n":
			return options[selected].Key, nil
		}
	}
}

func (f *terminalForm) key() (string, error) {
	key, _, err := f.in.ReadRune()
	if err != nil {
		return "", err
	}
	switch key {
	case 3:
		return "", errors.New("interrupted")
	case 4:
		return "", io.EOF
	case 27:
		sequence := string(key)
		for len(sequence) < 16 {
			part, _, err := f.in.ReadRune()
			if err != nil {
				return "", err
			}
			if part == 3 || part == 4 {
				return "", errors.New("interrupted")
			}
			sequence += string(part)
			if len(sequence) > 2 && part >= 0x40 && part <= 0x7e || len(sequence) == 2 && part != '[' && part != 'O' {
				break
			}
		}
		return sequence, nil
	default:
		return string(key), nil
	}
}

func (f *terminalForm) size() error {
	width, height, err := term.GetSize(f.fd)
	if err != nil {
		return err
	}
	if width < 8 || height < 8 {
		return fmt.Errorf("terminal is too small for guided input; enlarge it or use flags")
	}
	if f.width != 0 && (width != f.width || height != f.height) {
		// Reflow varies by terminal. Start below the old frame rather than erase shell history.
		if _, err := fmt.Fprint(f.out, "\r\n"); err != nil {
			return err
		}
		f.cursorRow = 0
	}
	f.width, f.height = width, height
	return nil
}

func (f *terminalForm) draw(controls []string, column int) error {
	width := f.width - 1
	var lines []string
	for _, line := range strings.Split(f.content(), "\n") {
		for uniseg.StringWidth(line) > width {
			part, rest := splitColumns(line, width)
			lines = append(lines, part)
			line = rest
		}
		lines = append(lines, line)
	}
	available := f.height - len(controls) - 2
	if len(lines) > available {
		available--
		f.page = min(f.page, (len(lines)-1)/available)
		start := f.page * available
		end := min(len(lines), start+available)
		lines = append(slices.Clone(lines[start:end]), fmt.Sprintf("  %d–%d of %d lines · PgUp/PgDn", start+1, end, len(lines)))
	}
	for _, control := range controls {
		lines = append(lines, control)
	}
	if column >= 0 {
		last := len(lines) - 1
		for column >= width {
			graphemes := uniseg.NewGraphemes(lines[last])
			if !graphemes.Next() {
				break
			}
			column -= graphemes.Width()
			lines[last] = lines[last][len(graphemes.Str()):]
		}
	}
	var output strings.Builder
	output.WriteString("\x1b[?25l\r")
	if f.cursorRow > 0 {
		fmt.Fprintf(&output, "\x1b[%dA", f.cursorRow)
	}
	output.WriteString("\x1b[J")
	for _, line := range lines {
		if uniseg.StringWidth(line) > width {
			prefix, _ := splitColumns(line, width-1)
			line = prefix + "…"
		}
		output.WriteString(line)
		output.WriteString("\r\n")
	}
	f.cursorRow = len(lines)
	if column >= 0 {
		output.WriteString("\x1b[1A\r")
		f.cursorRow--
		if column > 0 {
			fmt.Fprintf(&output, "\x1b[%dC", column)
		}
		output.WriteString("\x1b[?25h")
	}
	_, err := io.WriteString(f.out, output.String())
	return err
}

func splitColumns(text string, width int) (string, string) {
	graphemes := uniseg.NewGraphemes(text)
	end := 0
	for graphemes.Next() {
		width -= graphemes.Width()
		if width < 0 {
			break
		}
		_, end = graphemes.Positions()
	}
	return text[:end], text[end:]
}
