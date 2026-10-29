package guidance

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/rivo/uniseg"
	"github.com/unkeyed/unkey/pkg/prompt"
	"golang.org/x/term"
)

var errInterrupted = errors.New("interrupted")

const cancelHint = "Ctrl+C to cancel"

// palette keeps text in the 16 ANSI colors so terminal themes control contrast.
// Only the prompt marker uses the Unkey accent, because it never carries meaning alone.
type palette struct {
	color  bool
	accent string
}

func newPalette() palette {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CLICOLOR") == "0" || os.Getenv("TERM") == "dumb" {
		return palette{color: false, accent: ""}
	}
	accent := "\x1b[33m"
	switch {
	case os.Getenv("COLORTERM") == "truecolor" || os.Getenv("COLORTERM") == "24bit":
		accent = "\x1b[38;2;222;241;53m"
	case strings.Contains(os.Getenv("TERM"), "256color"):
		accent = "\x1b[38;5;191m"
	}
	return palette{color: true, accent: accent}
}

func (p palette) paint(code, text string) string {
	if !p.color || text == "" {
		return text
	}
	return code + text + "\x1b[0m"
}

func (p palette) mark(text string) string    { return p.paint(p.accent+"\x1b[1m", text) }
func (p palette) bold(text string) string    { return p.paint("\x1b[1m", text) }
func (p palette) value(text string) string   { return p.paint("\x1b[36m", text) }
func (p palette) done(text string) string    { return p.paint("\x1b[32m", text) }
func (p palette) hint(text string) string    { return p.paint("\x1b[90m", text) }
func (p palette) problem(text string) string { return p.paint("\x1b[31m", text) }

// separator keeps labels and values distinguishable when bold and color are unavailable.
func (p palette) separator() string {
	if p.color {
		return " "
	}
	return ": "
}

type section struct {
	path   string
	render func(palette) string
}

type terminalForm struct {
	in        *bufio.Reader
	out       *os.File
	fd        int
	palette   palette
	sections  []section
	notice    string
	pending   strings.Builder
	answered  bool
	selected  bool
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
	form.palette = newPalette()
	return &form
}

func (f *terminalForm) Write(data []byte) (int, error) {
	return f.pending.Write(data)
}

func (f *terminalForm) content() string {
	var content strings.Builder
	for _, section := range f.sections {
		content.WriteString(section.render(f.palette))
	}
	for _, text := range []string{f.notice, f.pending.String()} {
		for line := range strings.SplitSeq(strings.TrimRight(text, "\n"), "\n") {
			switch {
			case line == "":
			case strings.HasPrefix(line, " "):
				content.WriteString(line + "\n")
			default:
				content.WriteString(f.palette.problem("✗") + " " + strings.TrimPrefix(line, "✗ ") + "\n")
			}
		}
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
	f.pending.Reset()
	f.notice = ""
	return f.commit()
}

// commit leaves the current content in scrollback and starts the next frame below it.
func (f *terminalForm) commit() error {
	if err := f.size(); err != nil {
		return err
	}
	if err := f.draw(nil, -1, -1); err != nil {
		return err
	}
	f.cursorRow = 0
	f.page = 0
	return nil
}

func (f *terminalForm) footer(hints ...string) []string {
	if !f.answered {
		hints = append(hints, cancelHint)
	}
	if len(hints) == 0 {
		return nil
	}
	return []string{"  " + f.palette.hint(strings.Join(hints, " · "))}
}

func (f *terminalForm) String(label string, defaults ...string) (string, error) {
	defaultValue := ""
	if len(defaults) > 0 {
		defaultValue = defaults[0]
	}
	value, err := f.text(label, defaultValue, false)
	value = strings.TrimSpace(value)
	if value == "" {
		value = defaultValue
	}
	return value, err
}

func (f *terminalForm) Secret(label string) (string, error) {
	return f.text(label, "", true)
}

func (f *terminalForm) text(label, defaultValue string, hidden bool) (value string, err error) {
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
		line := f.palette.mark("?") + " " + f.palette.bold(label) + " "
		column := 3 + uniseg.StringWidth(label)
		if defaultValue != "" {
			shown, rest := splitColumns(defaultValue, max(1, f.width/2))
			if rest != "" {
				shown += "…"
			}
			line += f.palette.hint("("+shown+")") + " "
			column += uniseg.StringWidth(shown) + 3
		}
		var hints []string
		if hidden {
			hints = append(hints, "Input is hidden")
		} else {
			line += strings.ReplaceAll(strings.Join(input, ""), "\t", "⇥")
			column += uniseg.StringWidth(strings.ReplaceAll(strings.Join(input[:cursor], ""), "\t", "⇥"))
		}
		if err := f.draw(append([]string{line}, f.footer(hints...)...), 0, column); err != nil {
			return "", err
		}
		key, err := f.key()
		if err != nil {
			return "", err
		}
		switch key {
		case "\r", "\n":
			f.answered = true
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
		question := f.palette.mark("?") + " " + f.palette.bold(label)
		if !f.selected {
			question += "  " + f.palette.hint("↑/↓ to move · Enter to select")
		}
		lines := []string{question}
		for i := top; i < top+visible; i++ {
			if i == selected {
				lines = append(lines, f.palette.mark("❯")+" "+f.palette.value(options[i].Label))
			} else {
				lines = append(lines, "  "+options[i].Label)
			}
		}
		var hints []string
		if visible < len(options) {
			hints = append(hints, fmt.Sprintf("%d–%d of %d", top+1, top+visible, len(options)))
		}
		if err := f.draw(append(lines, f.footer(hints...)...), -1, -1); err != nil {
			return "", err
		}
		key, err := f.key()
		if err != nil {
			return "", err
		}
		switch key {
		case "\x1b[A", "\x10":
			selected = (selected + len(options) - 1) % len(options)
		case "\x1b[B", "\x0e":
			selected = (selected + 1) % len(options)
		case "\x1b[H", "\x1b[1~":
			selected = 0
		case "\x1b[F", "\x1b[4~":
			selected = len(options) - 1
		case "\x1b[5~":
			f.page = max(0, f.page-1)
		case "\x1b[6~":
			f.page++
		case "\r", "\n":
			f.answered, f.selected = true, true
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
		return "", errInterrupted
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
				return "", errInterrupted
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
		return fmt.Errorf("terminal is too small for prompts; enlarge it or pass flags")
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

// draw replaces the current frame. row and column place the cursor within controls; a negative row hides it.
func (f *terminalForm) draw(controls []string, row, column int) error {
	width := f.width - 1
	var lines []string
	if content := f.content(); content != "" {
		for _, line := range strings.Split(content, "\n") {
			for visibleWidth(line) > width {
				part, rest := wrapWords(line, width)
				lines = append(lines, part)
				line = "  " + rest
			}
			lines = append(lines, line)
		}
	}
	available := f.height - len(controls) - 2
	if len(lines) > available {
		available = max(1, available-1)
		f.page = min(f.page, (len(lines)-1)/available)
		start := f.page * available
		end := min(len(lines), start+available)
		lines = append(slices.Clone(lines[start:end]), "  "+f.palette.hint(fmt.Sprintf("%d–%d of %d lines · PgUp/PgDn", start+1, end, len(lines))))
	}
	target := -1
	if row >= 0 {
		target = len(lines) + row
	}
	lines = append(lines, controls...)
	if target >= 0 && column >= width {
		dropped, rest := splitColumns(lines[target], column-width+1)
		lines[target] = rest
		column -= visibleWidth(dropped)
	}
	var output strings.Builder
	output.WriteString("\x1b[?25l\r")
	if f.cursorRow > 0 {
		fmt.Fprintf(&output, "\x1b[%dA", f.cursorRow)
	}
	output.WriteString("\x1b[J")
	for _, line := range lines {
		if visibleWidth(line) > width {
			prefix, _ := splitColumns(line, width-1)
			line = prefix + "…"
		}
		output.WriteString(line)
		output.WriteString("\r\n")
	}
	f.cursorRow = len(lines)
	if target >= 0 {
		fmt.Fprintf(&output, "\x1b[%dA\r", len(lines)-target)
		f.cursorRow = target
		if column > 0 {
			fmt.Fprintf(&output, "\x1b[%dC", column)
		}
		output.WriteString("\x1b[?25h")
	}
	_, err := io.WriteString(f.out, output.String())
	return err
}

var (
	escapeSequence  = regexp.MustCompile(`^\x1b\[[0-9;?]*[ -/]*[@-~]`)
	escapeSequences = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
)

func visibleWidth(text string) int {
	return uniseg.StringWidth(escapeSequences.ReplaceAllString(text, ""))
}

// wrapWords cuts styled text at the last space within width, or after width columns if there is none.
func wrapWords(text string, width int) (string, string) {
	head, rest := splitColumns(text, width)
	plain := escapeSequences.ReplaceAllString(head, "")
	space := strings.LastIndex(plain, " ")
	if space < 0 || strings.TrimSpace(plain[:space]) == "" {
		return head, rest
	}
	head, rest = splitColumns(text, uniseg.StringWidth(plain[:space]))
	return head, strings.Replace(rest, " ", "", 1)
}

// splitColumns cuts styled text after width columns and carries any open style into the rest.
func splitColumns(text string, width int) (string, string) {
	var head strings.Builder
	style := ""
	rest := text
	for rest != "" {
		if sequence := escapeSequence.FindString(rest); sequence != "" {
			head.WriteString(sequence)
			if sequence == "\x1b[0m" {
				style = ""
			} else if strings.HasSuffix(sequence, "m") {
				style += sequence
			}
			rest = rest[len(sequence):]
			continue
		}
		cluster, remaining, cells, _ := uniseg.FirstGraphemeClusterInString(rest, -1)
		if cells > width {
			break
		}
		width -= cells
		head.WriteString(cluster)
		rest = remaining
	}
	if style != "" && rest != "" {
		return head.String() + "\x1b[0m", style + rest
	}
	return head.String(), rest
}

// lineForm asks one question per line without cursor movement, for terminals that cannot redraw.
type lineForm struct {
	in  *bufio.Reader
	out io.Writer
	fd  int
}

func newLineForm() *lineForm {
	return &lineForm{in: bufio.NewReader(os.Stdin), out: os.Stderr, fd: int(os.Stdin.Fd())}
}

func (f *lineForm) line() (string, error) {
	text, err := f.in.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func (f *lineForm) String(label string, defaults ...string) (string, error) {
	question := "? " + label
	if len(defaults) > 0 && defaults[0] != "" {
		question += " (" + defaults[0] + ")"
	}
	if _, err := fmt.Fprint(f.out, question+": "); err != nil {
		return "", err
	}
	value, err := f.line()
	if value == "" && len(defaults) > 0 {
		value = defaults[0]
	}
	return value, err
}

func (f *lineForm) Secret(label string) (string, error) {
	if _, err := fmt.Fprint(f.out, "? "+label+" (input is hidden): "); err != nil {
		return "", err
	}
	raw, err := term.ReadPassword(f.fd)
	_, writeErr := fmt.Fprintln(f.out)
	return string(raw), errors.Join(err, writeErr)
}

func (f *lineForm) SelectOrdered(label string, options []prompt.SelectOption, defaults ...string) (string, error) {
	selected := 0
	for i, option := range options {
		if len(defaults) > 0 && option.Key == defaults[0] {
			selected = i
		}
	}
	var menu strings.Builder
	fmt.Fprintf(&menu, "? %s\n", label)
	for i, option := range options {
		fmt.Fprintf(&menu, "  %d. %s\n", i+1, option.Label)
	}
	if _, err := io.WriteString(f.out, menu.String()); err != nil {
		return "", err
	}
	for {
		if _, err := fmt.Fprintf(f.out, "Enter a number between 1 and %d (default %d): ", len(options), selected+1); err != nil {
			return "", err
		}
		text, err := f.line()
		if err != nil {
			return "", err
		}
		if text == "" {
			return options[selected].Key, nil
		}
		if number, err := strconv.Atoi(text); err == nil && number >= 1 && number <= len(options) {
			return options[number-1].Key, nil
		}
	}
}
