package widget

import "strings"

const colGap = "  "

// StripANSI removes SGR escape sequences, leaving visible runes.
func StripANSI(s string) string {
	var b strings.Builder
	esc := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if s[i] == 'm' {
				esc = false
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// Width is the visible cell count: ANSI escapes are stripped and every
// remaining rune counts as one (all glyphs rendered here are single-width).
func Width(s string) int {
	return len([]rune(StripANSI(s)))
}

// Truncate shortens s to at most n visible cells, adding an ellipsis when it
// cuts (and dropping styling, since the tail is discarded).
func Truncate(s string, n int) string {
	if Width(s) <= n {
		return s
	}
	plain := []rune(StripANSI(s))
	if len(plain) <= n {
		return string(plain)
	}
	if n <= 1 {
		return string(plain[:n])
	}
	return string(plain[:n-1]) + "…"
}

// Pad pads s with spaces to w visible cells, or hard-cuts it (no ellipsis) if
// it is longer.
func Pad(line string, width int) string {
	if width <= 0 {
		return line
	}
	w := Width(line)
	if w > width {
		plain := []rune(StripANSI(line))
		if len(plain) > width {
			plain = plain[:width]
		}
		return string(plain)
	}
	return line + RepeatSpaces(width-w)
}

// PadPlain pads/ellipsizes a plain (unstyled) string to a fixed column width.
func PadPlain(s string, width int) string {
	plain := []rune(StripANSI(s))
	if width <= 0 {
		return string(plain)
	}
	if len(plain) > width {
		if width <= 1 {
			return string(plain[:width])
		}
		return string(plain[:width-1]) + "…"
	}
	return string(plain) + RepeatSpaces(width-len(plain))
}

func RepeatSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(" ", n)
}

func RepeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(string(r), n)
}

func EffectiveRowWidth(width int) int {
	w := width - 4
	if w < 20 {
		return 20
	}
	return w
}

// BarWidth is how far a selection bar extends: content width plus the marker,
// capped to the pane so it hugs the content instead of flooding the row.
func BarWidth(width, contentW int) int {
	w := contentW + 2
	if maxW := EffectiveRowWidth(width); w > maxW {
		return maxW
	}
	return w
}
