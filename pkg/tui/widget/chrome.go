package widget

import "strings"

// KeyHint renders a "key label" hint (bold key, dim label).
func (t Theme) KeyHint(key, label string) string {
	return t.Key.Bold(true).Render(key) + t.Dim.Render(" "+label)
}

// HeaderLine renders a title on the left and a right-aligned status string,
// filling width. right is expected to already be styled by the caller.
func (t Theme) HeaderLine(width int, title, right string) string {
	titled := t.Title.Render(title)
	gap := width - Width(titled) - Width(right)
	if gap < 1 {
		gap = 1
	}
	return titled + RepeatSpaces(gap) + right
}

// KeyBar renders a dim, single-line, width-filling hint bar (one column of left
// padding), truncating if the hints overflow.
func (t Theme) KeyBar(width int, hints ...string) string {
	line := strings.Join(hints, "  ")
	if Width(line) > width-2 && width > 2 {
		line = Truncate(line, width-2)
	}
	line = " " + line
	return line + RepeatSpaces(width-Width(line))
}

// ContextLine renders a breadcrumb ("seed · 4 workspaces") followed by a rule
// filling the width.
func (t Theme) ContextLine(width int, crumb, summary string) string {
	label := t.Accent.Render(crumb)
	if summary != "" {
		label += t.Dim.Render(" · " + summary)
	}
	label = " " + label + " "
	gap := width - Width(label)
	if gap < 0 {
		return Truncate(label, width)
	}
	return label + t.TableRule.Render(RepeatRune('─', gap))
}

// StatusLine renders a dim, truncated status message, or "" when empty.
func (t Theme) StatusLine(width int, msg string) string {
	if msg == "" {
		return ""
	}
	return t.Dim.Render(Truncate(msg, width))
}
