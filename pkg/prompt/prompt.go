package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Key codes for detecting user input in raw terminal mode.
// When the terminal is in raw mode, arrow keys send a 3-byte escape sequence:
//   - Byte 0: ESC (27 or 0x1B)
//   - Byte 1: '[' (91 or 0x5B)
//   - Byte 2: Direction code ('A'=up, 'B'=down, 'C'=right, 'D'=left)
//
// Regular keys send their ASCII value as a single byte.
const (
	keyUp    = 'A'  // Third byte of arrow up escape sequence: ESC [ A
	keyDown  = 'B'  // Third byte of arrow down escape sequence: ESC [ B
	keyEnter = '\r' // Carriage return (13), sent when Enter is pressed in raw mode
	keySpace = ' '  // Space character (32)
)

// Prompt manages interactive terminal prompts with configurable I/O.
type Prompt struct {
	in  io.Reader
	out io.Writer
	fd  int // file descriptor for raw mode, -1 if not a terminal
}

// Option configures a Prompt instance.
type Option func(*Prompt)

// WithReader sets the input reader. Defaults to os.Stdin.
func WithReader(r io.Reader) Option {
	return func(p *Prompt) {
		p.in = r
		if f, ok := r.(*os.File); ok {
			p.fd = int(f.Fd())
		} else {
			p.fd = -1
		}
	}
}

// WithWriter sets the output writer. Defaults to os.Stdout.
func WithWriter(w io.Writer) Option {
	return func(p *Prompt) {
		p.out = w
	}
}

// New creates a Prompt with the given options. Without options, uses os.Stdin
// and os.Stdout.
func New(opts ...Option) *Prompt {
	p := &Prompt{
		in:  os.Stdin,
		out: os.Stdout,
		fd:  int(os.Stdin.Fd()),
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// String prompts the user for text input and returns the trimmed response.
// The label is displayed followed by a colon and space. Input is read until
// the user presses Enter. If a default is provided and the user enters nothing,
// the default is returned.
func (p *Prompt) String(label string, defaultValue ...string) (string, error) {
	if len(defaultValue) > 0 && defaultValue[0] != "" {
		_, _ = fmt.Fprintf(p.out, "%s [%s]: ", label, defaultValue[0])
	} else {
		_, _ = fmt.Fprint(p.out, label+": ")
	}
	reader := bufio.NewReader(p.in)
	input, err := reader.ReadString('\n')
	result := strings.TrimSpace(input)
	if result == "" && len(defaultValue) > 0 {
		return defaultValue[0], err
	}
	return result, err
}

// Int prompts the user for an integer value. If a default is provided and the
// user enters nothing, the default is returned.
//
// Accepts human-readable suffixes for large numbers:
//   - k, K: thousand (1k = 1,000)
//   - m, M: million (1m = 1,000,000)
//   - b, B: billion (1b = 1,000,000,000)
//   - t, T: trillion (1t = 1,000,000,000,000)
//
// Decimal values are supported with suffixes if the result is a whole number:
// "1.5k" = 1500 (valid), "1.5" without suffix = error.
//
// Returns an error if the input cannot be parsed, is not a whole number, or would overflow.
func (p *Prompt) Int(label string, defaultValue ...int) (int, error) {
	var s string
	var err error
	if len(defaultValue) > 0 {
		s, err = p.String(label, strconv.Itoa(defaultValue[0]))
	} else {
		s, err = p.String(label)
	}
	if err != nil {
		return 0, err
	}
	return parseHumanInt(s)
}

// Float prompts the user for a floating-point value. If a default is provided
// and the user enters nothing, the default is returned.
//
// Accepts human-readable suffixes for large numbers:
//   - k, K: thousand (1.5k = 1,500.0)
//   - m, M: million (2.5m = 2,500,000.0)
//   - b, B: billion (1b = 1,000,000,000.0)
//   - t, T: trillion (1t = 1,000,000,000,000.0)
//
// Returns an error if the input cannot be parsed.
func (p *Prompt) Float(label string, defaultValue ...float64) (float64, error) {
	var s string
	var err error
	if len(defaultValue) > 0 {
		s, err = p.String(label, strconv.FormatFloat(defaultValue[0], 'f', -1, 64))
	} else {
		s, err = p.String(label)
	}
	if err != nil {
		return 0, err
	}
	return parseHumanFloat(s)
}

// Select displays a single-choice menu and returns the selected key.
// The options map keys are internal codes returned on selection; values are display labels.
// The user navigates with arrow keys and confirms with Enter.
// If a default key is provided, that option is pre-selected.
//
// Returns an error if the terminal cannot be switched to raw mode (e.g., stdin
// is not a TTY) or if the user interrupts with Ctrl+C.
func (p *Prompt) Select(label string, options map[string]string, defaultKey ...string) (string, error) {
	// Switch terminal to raw mode to read individual keypresses without waiting for Enter.
	// Raw mode disables line buffering and echo, giving us direct access to input.
	// We must restore the original state when done, even on error.
	oldState, err := term.MakeRaw(p.fd)
	if err != nil {
		return "", err
	}
	defer func() { _ = term.Restore(p.fd, oldState) }()

	// Extract keys into a slice for consistent ordering during iteration.
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}

	// Find the index of the default key, if provided.
	selected := 0
	if len(defaultKey) > 0 {
		for i, k := range keys {
			if k == defaultKey[0] {
				selected = i
				break
			}
		}
	}

	// render draws the current menu state to the terminal.
	// It first hides the cursor to prevent flickering, then draws all options,
	// and finally moves the cursor back up to the start position for the next redraw.
	render := func() {
		_, _ = fmt.Fprint(p.out, hideCursor)
		_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+label+newLine)
		for i, key := range keys {
			displayLabel := options[key]
			if i == selected {
				_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+"  "+colorCyan+pointerSymbol+" "+displayLabel+colorReset+newLine)
			} else {
				_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+"    "+displayLabel+newLine)
			}
		}
		// Move cursor back up to the start of the menu for the next render cycle.
		// We move up len(keys)+1 lines: one for each option plus the label line.
		_, _ = fmt.Fprint(p.out, cursorUp(len(keys)+1))
	}

	render()

	// Read input in a loop. We use a 3-byte buffer because arrow keys send
	// a 3-byte escape sequence (ESC [ A/B/C/D), while regular keys use 1 byte.
	buf := make([]byte, 3)
	for {
		n, err := p.in.Read(buf)
		if err != nil {
			return "", err
		}

		// Check for arrow key escape sequences: ESC (27) + '[' (91) + direction.
		// n==3 ensures we received a complete escape sequence.
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case keyUp:
				// Wrap around: if at top, go to bottom.
				selected = (selected - 1 + len(keys)) % len(keys)
			case keyDown:
				// Wrap around: if at bottom, go to top.
				selected = (selected + 1) % len(keys)
			}
		} else if buf[0] == keyEnter || buf[0] == '\n' {
			// User confirmed selection. Move cursor below the menu and restore visibility.
			_, _ = fmt.Fprint(p.out, cursorDown(len(keys)+1))
			_, _ = fmt.Fprint(p.out, showCursor)
			return keys[selected], nil
		} else if buf[0] == 3 {
			// Ctrl+C sends byte 3 (ETX - End of Text). Treat as interrupt.
			_, _ = fmt.Fprint(p.out, showCursor)
			return "", fmt.Errorf("interrupted")
		}
		render()
	}
}

// MultiSelect displays a multi-choice menu and returns the selected keys.
// The options map keys are internal codes returned on selection; values are display labels.
// The user navigates with arrow keys, toggles selection with Space, and confirms with Enter.
// If default keys are provided, those options are pre-selected.
//
// Returns an error if the terminal cannot be switched to raw mode (e.g., stdin
// is not a TTY) or if the user interrupts with Ctrl+C.
func (p *Prompt) MultiSelect(label string, options map[string]string, defaultKeys ...string) ([]string, error) {
	// Switch terminal to raw mode to read individual keypresses without waiting for Enter.
	// Raw mode disables line buffering and echo, giving us direct access to input.
	// We must restore the original state when done, even on error.
	oldState, err := term.MakeRaw(p.fd)
	if err != nil {
		return nil, err
	}
	defer func() { _ = term.Restore(p.fd, oldState) }()

	// Extract keys into a slice for consistent ordering during iteration.
	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}

	// Build a set of default keys for O(1) lookup when initializing selections.
	defaultSet := make(map[string]bool)
	for _, k := range defaultKeys {
		defaultSet[k] = true
	}

	// cursor tracks which option the pointer (❯) is on.
	// selected tracks which options have been toggled on (checkmark visible).
	cursor := 0
	selected := make(map[int]bool)
	for i, key := range keys {
		if defaultSet[key] {
			selected[i] = true
		}
	}

	// render draws the current menu state to the terminal.
	// It first hides the cursor to prevent flickering, then draws all options
	// with their selection state, and finally moves the cursor back up for the next redraw.
	render := func() {
		_, _ = fmt.Fprint(p.out, hideCursor)
		_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+label+" (space to select, enter to confirm)"+newLine)
		for i, key := range keys {
			// Show filled circle (●) for selected items, empty circle (○) for unselected.
			check := unselectedSymbol
			if selected[i] {
				check = colorGreen + selectedSymbol + colorReset
			}
			displayLabel := options[key]
			// Show pointer (❯) next to the currently focused option.
			if i == cursor {
				_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+"  "+colorCyan+pointerSymbol+colorReset+" "+check+" "+displayLabel+newLine)
			} else {
				_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+"    "+check+" "+displayLabel+newLine)
			}
		}
		// Move cursor back up to the start of the menu for the next render cycle.
		// We move up len(keys)+1 lines: one for each option plus the label line.
		_, _ = fmt.Fprint(p.out, cursorUp(len(keys)+1))
	}

	render()

	// Read input in a loop. We use a 3-byte buffer because arrow keys send
	// a 3-byte escape sequence (ESC [ A/B/C/D), while regular keys use 1 byte.
	buf := make([]byte, 3)
	for {
		n, err := p.in.Read(buf)
		if err != nil {
			return nil, err
		}

		// Check for arrow key escape sequences: ESC (27) + '[' (91) + direction.
		// n==3 ensures we received a complete escape sequence.
		if n == 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case keyUp:
				// Wrap around: if at top, go to bottom.
				cursor = (cursor - 1 + len(keys)) % len(keys)
			case keyDown:
				// Wrap around: if at bottom, go to top.
				cursor = (cursor + 1) % len(keys)
			}
		} else if buf[0] == keySpace {
			// Toggle selection state of the currently focused option.
			selected[cursor] = !selected[cursor]
		} else if buf[0] == keyEnter || buf[0] == '\n' {
			// User confirmed selections. Move cursor below the menu and restore visibility.
			_, _ = fmt.Fprint(p.out, cursorDown(len(keys)+1))
			_, _ = fmt.Fprint(p.out, showCursor)
			// Collect all selected keys in order.
			var result []string
			for i, key := range keys {
				if selected[i] {
					result = append(result, key)
				}
			}
			return result, nil
		} else if buf[0] == 3 {
			// Ctrl+C sends byte 3 (ETX - End of Text). Treat as interrupt.
			_, _ = fmt.Fprint(p.out, showCursor)
			return nil, fmt.Errorf("interrupted")
		}
		render()
	}
}
