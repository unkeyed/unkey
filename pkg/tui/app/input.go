package app

import (
	"os"
	"unicode/utf8"
)

// ANSI/control sequences the runtime writes.
const (
	enterAltScreen   = "\033[?1049h"
	exitAltScreen    = "\033[?1049l"
	hideCursor       = "\033[?25l"
	showCursor       = "\033[?25h"
	cursorHome       = "\033[H"
	clearLineToEnd   = "\033[K"
	clearScreenBelow = "\033[J"
	// disableWrap turns off autowrap (DECAWM). A full-width line otherwise
	// triggers the terminal's last-column wrap/defer behavior, which corrupts
	// the frame (a trailing character bleeds onto the next row).
	disableWrap = "\033[?7l"
	enableWrap  = "\033[?7h"
)

// readInput reads the raw tty and posts parsed key events until the reader
// errors (e.g. on shutdown).
func readInput(in *os.File, msgCh chan<- Msg) {
	buf := make([]byte, 256)
	for {
		n, err := in.Read(buf)
		if n > 0 {
			for _, k := range parseKeys(buf[:n]) {
				msgCh <- k
			}
		}
		if err != nil {
			return
		}
	}
}

// parseKeys decodes a raw input buffer into key events. Escape sequences are
// assumed to arrive whole within a single read, which holds for terminals in
// raw mode (the same assumption pkg/prompt relies on).
func parseKeys(b []byte) []KeyMsg {
	var keys []KeyMsg
	for i := 0; i < len(b); {
		c := b[i]
		switch {
		case c == 27: // ESC: alone, or the start of a CSI sequence
			if i+2 < len(b) && b[i+1] == '[' {
				switch b[i+2] {
				case 'A':
					keys = append(keys, KeyMsg{Type: KeyUp}) //nolint:exhaustruct
				case 'B':
					keys = append(keys, KeyMsg{Type: KeyDown}) //nolint:exhaustruct
				case 'C':
					keys = append(keys, KeyMsg{Type: KeyRight}) //nolint:exhaustruct
				case 'D':
					keys = append(keys, KeyMsg{Type: KeyLeft}) //nolint:exhaustruct
				case 'Z':
					keys = append(keys, KeyMsg{Type: KeyShiftTab}) //nolint:exhaustruct
				}
				i += 3
				continue
			}
			keys = append(keys, KeyMsg{Type: KeyEsc}) //nolint:exhaustruct
			i++
		case c == 13 || c == 10: // CR / LF
			keys = append(keys, KeyMsg{Type: KeyEnter}) //nolint:exhaustruct
			i++
		case c == 9:
			keys = append(keys, KeyMsg{Type: KeyTab}) //nolint:exhaustruct
			i++
		case c == 3:
			keys = append(keys, KeyMsg{Type: KeyCtrlC}) //nolint:exhaustruct
			i++
		case c == 16:
			keys = append(keys, KeyMsg{Type: KeyCtrlP}) //nolint:exhaustruct
			i++
		case c == 14:
			keys = append(keys, KeyMsg{Type: KeyCtrlN}) //nolint:exhaustruct
			i++
		case c == 127 || c == 8:
			keys = append(keys, KeyMsg{Type: KeyBackspace}) //nolint:exhaustruct
			i++
		case c < 32:
			// Other control bytes: ignore.
			i++
		default:
			r, size := utf8.DecodeRune(b[i:])
			if r == utf8.RuneError && size <= 1 {
				i++
				continue
			}
			keys = append(keys, KeyMsg{Type: KeyRunes, Runes: []rune{r}}) //nolint:exhaustruct
			i += size
		}
	}
	return keys
}
