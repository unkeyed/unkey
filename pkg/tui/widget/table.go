package widget

import "strings"

// Column is one table column: a header title, a fixed width, and whether it
// renders muted on unselected rows (for IDs/metadata).
type Column struct {
	Title string
	Width int
	Dim   bool
}

// MenuItem is one row of a name + description menu.
type MenuItem struct {
	Name string
	Desc string
}

func tableRowWidth(cols []Column) int {
	if len(cols) == 0 {
		return 0
	}
	w := 0
	for i, c := range cols {
		w += c.Width
		if i > 0 {
			w += len(colGap)
		}
	}
	return w
}

func (t Theme) tableHeader(cols []Column) string {
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = PadPlain(strings.ToUpper(c.Title), c.Width)
	}
	line := "  " + strings.Join(parts, colGap)
	rule := "  " + RepeatRune('─', Width(line)-2)
	return t.TableHeader.Render(line) + "\n" + t.TableRule.Render(rule) + "\n"
}

func (t Theme) tableBody(width int, cols []Column, rows [][]string, cursor int) string {
	if len(rows) == 0 {
		return ""
	}
	var b strings.Builder
	rowW := BarWidth(width, tableRowWidth(cols))
	for i, cells := range rows {
		selected := i == cursor
		parts := make([]string, len(cols))
		for j, c := range cols {
			val := ""
			if j < len(cells) {
				val = cells[j]
			}
			cell := PadPlain(val, c.Width)
			if !selected && c.Dim {
				cell = t.Dim.Render(cell)
			}
			parts[j] = cell
		}
		if selected {
			line := Pad("▸ "+strings.Join(parts, colGap), rowW)
			b.WriteString(t.SelectedRow.Render(line))
		} else {
			b.WriteString("  " + strings.Join(parts, colGap))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// Table renders a header + rows with the cursor row highlighted.
func (t Theme) Table(width int, cols []Column, rows [][]string, cursor int) string {
	return t.tableHeader(cols) + t.tableBody(width, cols, rows, cursor)
}

// MenuList renders a name/description list with the cursor row highlighted.
func (t Theme) MenuList(width int, title string, items []MenuItem, cursor int) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(t.SectionTitle(title))
		b.WriteString("\n")
	}
	nameW := 22
	descW := 36
	rowW := BarWidth(width, nameW+len(colGap)+descW)
	for i, item := range items {
		selected := i == cursor
		name := PadPlain(item.Name, nameW)
		desc := PadPlain(item.Desc, descW)
		if selected {
			line := Pad("▸ "+name+colGap+desc, rowW)
			b.WriteString(t.SelectedRow.Render(line))
		} else {
			b.WriteString("  " + name + colGap + t.Dim.Render(desc))
		}
		b.WriteString("\n")
	}
	return b.String()
}
