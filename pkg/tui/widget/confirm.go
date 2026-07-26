package widget

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

// ConfirmLine is one row on a pre-action confirm screen.
type ConfirmLine struct {
	Label      string
	Value      string
	Scope      bool // Enter re-opens the picker for this field
	Adjustable bool // ←/→ steps this value
}

// ConfirmNav is the decoded result of a key on a confirm screen.
type ConfirmNav struct {
	Cursor int
	Esc    bool
	Enter  bool
	Adjust int
}

const confirmLabelWidth = 12

func (t Theme) ConfirmScreen(width int, title, subtitle string, lines []ConfirmLine, cursor int) string {
	var b strings.Builder
	b.WriteString(t.Header.Render(title))
	if subtitle != "" {
		b.WriteString("  " + t.Dim.Render(subtitle))
	}
	b.WriteString("\n\n")

	valueW := 28
	rowW := BarWidth(width, confirmLabelWidth+1+valueW)
	for i, row := range lines {
		selected := i == cursor
		value := row.Value
		if selected && row.Adjustable {
			value = "‹ " + value + " ›"
		}
		label := PadPlain(row.Label, confirmLabelWidth)
		if selected {
			line := Pad("▸ "+label+" "+PadPlain(value, valueW), rowW)
			b.WriteString(t.SelectedRow.Render(line))
		} else {
			b.WriteString("  " + t.Dim.Render(label) + " " + PadPlain(value, valueW))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(t.Dim.Render(confirmFooter(lines, cursor)))
	return b.String()
}

func confirmFooter(lines []ConfirmLine, cursor int) string {
	if cursor < len(lines) && lines[cursor].Scope {
		return "Enter change   Esc back"
	}
	if confirmHasAdjustable(lines) {
		return "←/→ adjust   Enter run   Esc back"
	}
	return "Enter run   Esc back"
}

func (t Theme) ConfirmKeyHints(lines []ConfirmLine, cursor int) []string {
	if cursor < len(lines) && lines[cursor].Scope {
		return []string{
			t.KeyHint("↑/↓", "nav"),
			t.KeyHint("enter", "change"),
			t.KeyHint("esc", "back"),
		}
	}
	if confirmHasAdjustable(lines) {
		return []string{
			t.KeyHint("↑/↓", "nav"),
			t.KeyHint("←/→", "adjust"),
			t.KeyHint("enter", "run"),
			t.KeyHint("esc", "back"),
		}
	}
	return []string{
		t.KeyHint("↑/↓", "nav"),
		t.KeyHint("enter", "run"),
		t.KeyHint("esc", "back"),
	}
}

func confirmHasAdjustable(lines []ConfirmLine) bool {
	for _, line := range lines {
		if line.Adjustable {
			return true
		}
	}
	return false
}

// ConfirmFirstAdjustable is the index of the first adjustable row, or 0.
func ConfirmFirstAdjustable(lines []ConfirmLine) int {
	for i, line := range lines {
		if line.Adjustable {
			return i
		}
	}
	return 0
}

// NavigateConfirm decodes a key against the confirm rows.
func NavigateConfirm(key app.KeyMsg, cursor int, lines []ConfirmLine) ConfirmNav {
	res := ConfirmNav{Cursor: cursor} //nolint:exhaustruct
	if key.IsEsc() {
		res.Esc = true
		return res
	}
	switch {
	case key.IsUp():
		res.Cursor = app.MoveCursor(cursor, -1, len(lines))
	case key.IsDown():
		res.Cursor = app.MoveCursor(cursor, 1, len(lines))
	case key.IsLeft():
		if cursor < len(lines) && lines[cursor].Adjustable {
			res.Adjust = -1
		}
	case key.IsRight():
		if cursor < len(lines) && lines[cursor].Adjustable {
			res.Adjust = 1
		}
	case key.IsEnter():
		res.Enter = true
	}
	return res
}

// HandleConfirmEnter classifies an Enter: change a Scope field, or run.
func HandleConfirmEnter(nav ConfirmNav, lines []ConfirmLine) (changeScope bool, run bool) {
	if !nav.Enter {
		return false, false
	}
	if nav.Cursor < len(lines) && lines[nav.Cursor].Scope {
		return true, false
	}
	return false, true
}
