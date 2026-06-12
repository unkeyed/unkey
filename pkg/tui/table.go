package tui

import "strings"

// Table accumulates rows and prints them with columns aligned to the widest
// cell. Column widths account for ANSI escape codes, so styled cells (for
// example a Green status) align with plain ones.
type Table struct {
	r       *Renderer
	headers []string
	rows    [][]string
	indent  int
}

// Table starts a table. Headers are optional; pass none for a bare aligned
// grid. Headers render dim so data stands out over chrome.
func (r *Renderer) Table(headers ...string) *Table {
	return &Table{r: r, headers: headers, rows: nil, indent: 0}
}

// Indent shifts the whole table right by n spaces.
func (t *Table) Indent(n int) *Table {
	t.indent = n
	return t
}

// Row appends a row. Rows may have more or fewer cells than the header.
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// Print writes the table in a single write. A table with headers but no rows
// prints nothing: the caller decides how to phrase emptiness.
func (t *Table) Print() {
	if len(t.rows) == 0 {
		return
	}

	var widths []int
	measure := func(cells []string) {
		for i, cell := range cells {
			for len(widths) <= i {
				widths = append(widths, 0)
			}
			if w := visibleWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}
	measure(t.headers)
	for _, row := range t.rows {
		measure(row)
	}

	var b strings.Builder
	if len(t.headers) > 0 {
		styled := make([]string, len(t.headers))
		for i, h := range t.headers {
			styled[i] = t.r.Dim(h)
		}
		t.writeRow(&b, styled, widths)
	}
	for _, row := range t.rows {
		t.writeRow(&b, row, widths)
	}
	t.r.write(b.String())
}

func (t *Table) writeRow(b *strings.Builder, cells []string, widths []int) {
	writeSpaces(b, t.indent)
	for i, cell := range cells {
		if i == len(cells)-1 {
			b.WriteString(cell) // never pad the last cell: no trailing whitespace
			break
		}
		writePadded(b, cell, widths[i]+2) // two-space column gap
	}
	b.WriteByte('\n')
}
