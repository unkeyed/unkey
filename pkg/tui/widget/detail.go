package widget

import "strings"

// SplitThreshold is the terminal width at or above which a list+detail split is
// used; below it the list takes the full width and the detail panel is hidden.
const SplitThreshold = 100

// DetailOuter is the outer width (border included) of the detail box.
const DetailOuter = 36

// SplitWidths returns the outer box widths for the list and detail columns and
// whether to split at this terminal width. The boxes tile the full width.
func SplitWidths(termWidth int) (listOuter, detOuter int, split bool) {
	if termWidth < SplitThreshold {
		return termWidth, 0, false
	}
	listOuter = termWidth - DetailOuter
	if listOuter < 60 {
		return termWidth, 0, false
	}
	return listOuter, DetailOuter, true
}

// DetailStyle selects the color of a field value.
type DetailStyle int

const (
	DetailPlain DetailStyle = iota
	DetailOK
	DetailMid
	DetailBad
	DetailMuted
)

// Field is one label/value row in a detail panel.
type Field struct {
	Label string
	Value string
	Style DetailStyle
}

// Detail draws a titled label/value panel sized to content width w (the box
// inner width; the frame adds border + padding around it).
func (t Theme) Detail(w int, title, subtitle string, fields []Field) string {
	textW := w - 4
	if textW < 8 {
		textW = 8
	}
	const labelW = 9

	var b strings.Builder
	b.WriteString(t.Header.Render(Truncate(title, textW)))
	if subtitle != "" {
		b.WriteString("\n")
		b.WriteString(t.Dim.Render(Truncate(subtitle, textW)))
	}
	b.WriteString("\n\n")
	valueW := textW - labelW - 1
	if valueW < 4 {
		valueW = 4
	}
	for _, f := range fields {
		label := t.DetailLabel.Render(PadPlain(f.Label, labelW))
		value := t.detailValue(f.Style, wrapValue(f.Value, valueW, labelW+1))
		b.WriteString(label + " " + value)
		b.WriteString("\n")
	}
	return b.String()
}

func (t Theme) detailValue(s DetailStyle, v string) string {
	switch s {
	case DetailOK:
		return t.OK.Render(v)
	case DetailMid:
		return t.Mid.Render(v)
	case DetailBad:
		return t.Err.Render(v)
	case DetailMuted:
		return t.Dim.Render(v)
	case DetailPlain:
		return t.DetailValue.Render(v)
	default:
		return t.DetailValue.Render(v)
	}
}

// OrDash returns an em dash for an empty string.
func OrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// wrapValue hard-wraps a long value across lines, indenting continuation lines
// under the value column so full IDs stay readable instead of being truncated.
func wrapValue(v string, width, indent int) string {
	if width < 4 {
		width = 4
	}
	runes := []rune(v)
	if len(runes) <= width {
		return v
	}
	var lines []string
	for len(runes) > width {
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	lines = append(lines, string(runes))
	return strings.Join(lines, "\n"+RepeatSpaces(indent))
}
