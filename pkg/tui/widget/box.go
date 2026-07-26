package widget

import "strings"

// Box draws a single-border box exactly width wide and height tall, with body
// inset by one column of padding. Content taller or wider than the frame is
// clipped so surrounding chrome stays on screen.
func (t Theme) Box(width, height int, body string) string {
	if width < 20 {
		width = 20
	}
	if height < 4 {
		height = 4
	}
	innerW := width - 2 // columns between the vertical borders
	if innerW < 1 {
		innerW = 1
	}
	innerH := height - 2 // rows between top and bottom borders
	if innerH < 1 {
		innerH = 1
	}
	textW := innerW - 2 // one column of padding on each side
	if textW < 1 {
		textW = 1
	}

	lines := strings.Split(body, "\n")
	var b strings.Builder
	b.WriteString(t.Border.Render("┌" + strings.Repeat("─", innerW) + "┐"))
	b.WriteString("\n")
	for i := 0; i < innerH; i++ {
		content := ""
		if i < len(lines) {
			content = lines[i]
		}
		b.WriteString(t.Border.Render("│"))
		b.WriteString(" " + Pad(content, textW) + " ")
		b.WriteString(t.Border.Render("│"))
		b.WriteString("\n")
	}
	b.WriteString(t.Border.Render("└" + strings.Repeat("─", innerW) + "┘"))
	return b.String()
}

// JoinHorizontal places two multi-line blocks side by side, padding the left
// block's lines to its widest so the right block starts at a fixed column.
func JoinHorizontal(left, right string) string {
	l := strings.Split(left, "\n")
	r := strings.Split(right, "\n")
	leftW := 0
	for _, ln := range l {
		leftW = max(leftW, Width(ln))
	}
	rows := max(len(l), len(r))
	var b strings.Builder
	for i := range rows {
		ll, rr := "", ""
		if i < len(l) {
			ll = l[i]
		}
		if i < len(r) {
			rr = r[i]
		}
		b.WriteString(ll)
		b.WriteString(RepeatSpaces(leftW - Width(ll)))
		b.WriteString(rr)
		if i < rows-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// SectionTitle renders a bold dim heading with a trailing blank line; callers
// add their own newline before the section body.
func (t Theme) SectionTitle(s string) string {
	return t.Dim.Bold(true).Render(s) + "\n"
}
