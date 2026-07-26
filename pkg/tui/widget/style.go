// Package widget is a reusable, theme-driven terminal-UI widget kit: tables,
// menus, confirm screens, detail panels, tabs, box frames, and the text/width
// helpers they need. Widgets are methods on a Theme so colors are injected by
// the caller, not hardcoded. Built on the standard library plus
// github.com/unkeyed/unkey/pkg/tui/app (for key events).
package widget

import (
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// DetectColor decides whether to emit color: off for NO_COLOR, TERM=dumb, or a
// non-tty stdout.
func DetectColor() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Style is a minimal 256-color styler: a foreground/background palette index
// plus bold, emitted as SGR escape codes. A zero-value Style renders plain.
type Style struct {
	fg   int // -1 = unset
	bg   int // -1 = unset
	bold bool
	on   bool
}

func (s Style) Foreground(code int) Style {
	s.fg = code
	return s
}

func (s Style) Background(code int) Style {
	s.bg = code
	return s
}

func (s Style) Bold(on bool) Style {
	s.bold = on
	return s
}

func (s Style) Render(str string) string {
	if !s.on {
		return str
	}
	var codes []string
	if s.bold {
		codes = append(codes, "1")
	}
	if s.fg >= 0 {
		codes = append(codes, "38;5;"+strconv.Itoa(s.fg))
	}
	if s.bg >= 0 {
		codes = append(codes, "48;5;"+strconv.Itoa(s.bg))
	}
	if len(codes) == 0 {
		return str
	}
	return "\033[" + strings.Join(codes, ";") + "m" + str + "\033[0m"
}
