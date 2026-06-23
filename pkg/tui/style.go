package tui

import "strings"

// SGR (Select Graphic Rendition) escape codes, same dialect as pkg/prompt.
const (
	codeReset  = "\033[0m"
	codeBold   = "\033[1m"
	codeDim    = "\033[90m" // bright black reads as gray on light and dark themes
	codeRed    = "\033[31m"
	codeGreen  = "\033[32m"
	codeYellow = "\033[33m"
	codeCyan   = "\033[36m"
)

func (r *Renderer) paint(code, s string) string {
	if !r.color || s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(code) + len(s) + len(codeReset))
	b.WriteString(code)
	b.WriteString(s)
	b.WriteString(codeReset)
	return b.String()
}

// Bold renders s bold when color is enabled.
func (r *Renderer) Bold(s string) string { return r.paint(codeBold, s) }

// Dim renders s gray when color is enabled. Use for secondary detail such as
// ids, labels, and hints.
func (r *Renderer) Dim(s string) string { return r.paint(codeDim, s) }

// Red renders s red when color is enabled. Use for failure states.
func (r *Renderer) Red(s string) string { return r.paint(codeRed, s) }

// Green renders s green when color is enabled. Use for success states.
func (r *Renderer) Green(s string) string { return r.paint(codeGreen, s) }

// Yellow renders s yellow when color is enabled. Use for pending or
// in-progress states.
func (r *Renderer) Yellow(s string) string { return r.paint(codeYellow, s) }

// Cyan renders s cyan when color is enabled. Use for links and identifiers
// the user is expected to act on.
func (r *Renderer) Cyan(s string) string { return r.paint(codeCyan, s) }
