package widget

import "strings"

// TabBar renders a row of labels with the active one highlighted, plus an
// underline that marks the active tab. Labels are pre-formatted (e.g. "1 Seed")
// and the whole line fills width.
func (t Theme) TabBar(width int, labels []string, active int) (line, underline string) {
	lbls := make([]string, 0, len(labels))
	marks := make([]string, 0, len(labels))
	for i, label := range labels {
		cellW := Width(label) + 2
		if i == active {
			lbls = append(lbls, t.TabActive.Render(" "+label+" "))
			marks = append(marks, t.Accent.Render(RepeatRune('─', cellW)))
			continue
		}
		lbls = append(lbls, t.TabInactive.Render(" "+label+" "))
		marks = append(marks, RepeatSpaces(cellW))
	}
	joined := strings.Join(lbls, "")
	line = joined + RepeatSpaces(width-Width(joined))
	if active >= 0 && active < len(marks) {
		underline = t.Border.Render(strings.Join(marks, ""))
	}
	return line, underline
}
