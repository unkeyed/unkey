package tui

import "strings"

// visibleWidth returns the number of terminal cells s occupies, skipping ANSI
// escape sequences so styled cells align with plain ones. CSI sequences run
// from ESC to the first ASCII letter.
func visibleWidth(s string) int {
	width := 0
	inEscape := false
	for _, ch := range s {
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		if ch == '\033' {
			inEscape = true
			continue
		}
		width++
	}
	return width
}

// writePadded writes s to b followed by enough spaces to fill width terminal
// cells, measuring visible width so ANSI styling does not break alignment.
// Writing into the builder avoids the intermediate string a pad-then-append
// approach would allocate per cell.
func writePadded(b *strings.Builder, s string, width int) {
	b.WriteString(s)
	writeSpaces(b, width-visibleWidth(s))
}

const spaces = "                                                                "

// writeSpaces writes n spaces to b without allocating, chunking through a
// fixed buffer for the rare n larger than it.
func writeSpaces(b *strings.Builder, n int) {
	for n > 0 {
		chunk := n
		if chunk > len(spaces) {
			chunk = len(spaces)
		}
		b.WriteString(spaces[:chunk])
		n -= chunk
	}
}
