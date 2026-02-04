package prompt

// ANSI escape sequences for terminal control.
//
// These codes follow the ANSI X3.64 / ECMA-48 standard and work on most modern
// terminals (macOS Terminal, iTerm2, Windows Terminal, Linux terminals).
//
// Escape sequences start with ESC (0x1B or \033 in octal), followed by '[' (CSI - Control
// Sequence Introducer), then parameters and a command letter.
//
// Format: ESC [ <params> <command>
//
// References:
//   - https://en.wikipedia.org/wiki/ANSI_escape_code
//   - https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
const (
	// Cursor visibility controls.
	// These use DEC Private Mode sequences (ESC [ ? <mode> h/l).
	// Mode 25 controls cursor visibility: 'l' = hide, 'h' = show.
	hideCursor = "\033[?25l" // ESC [ ? 25 l - hide cursor
	showCursor = "\033[?25h" // ESC [ ? 25 h - show cursor

	// Line editing.
	// clearLine erases from cursor to end of line (ESC [ K or ESC [ 0 K).
	// We combine with carriageReturn to clear the entire line.
	clearLine      = "\033[K" // ESC [ K - erase from cursor to end of line
	carriageReturn = "\r"     // Move cursor to beginning of current line (ASCII 13)
	newLine        = "\r\n"   // Carriage return + line feed for new line in raw mode

	// SGR (Select Graphic Rendition) sequences for text colors.
	// Format: ESC [ <color-code> m
	// 30-37 are foreground colors, 0 resets all attributes.
	colorCyan  = "\033[36m" // ESC [ 36 m - cyan foreground (color code 36)
	colorGreen = "\033[32m" // ESC [ 32 m - green foreground (color code 32)
	colorRed   = "\033[31m" // ESC [ 31 m - red foreground (color code 31)
	colorDim   = "\033[90m" // ESC [ 90 m - bright black (gray) foreground
	colorReset = "\033[0m"  // ESC [ 0 m - reset all text attributes to default

	// Unicode symbols for visual indicators.
	pointerSymbol    = "❯" // Shows which option the cursor is on
	selectedSymbol   = "●" // Filled circle for selected items in MultiSelect
	unselectedSymbol = "○" // Empty circle for unselected items in MultiSelect
)

// cursorUp returns an ANSI escape sequence to move the cursor up n lines.
// Used to reposition the cursor at the start of the menu after rendering,
// allowing us to redraw the menu in place without scrolling.
// Format: ESC [ <n> A
func cursorUp(n int) string {
	return "\033[" + itoa(n) + "A"
}

// cursorDown returns an ANSI escape sequence to move the cursor down n lines.
// Used to move past the menu when the user confirms their selection.
// Format: ESC [ <n> B
func cursorDown(n int) string {
	return "\033[" + itoa(n) + "B"
}

// itoa converts a small positive integer to its string representation.
// We use a simple recursive implementation to avoid importing strconv
// for this single use case. Only used for cursor movement counts,
// which are typically small numbers (< 100).
func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
