package ui

import "github.com/unkeyed/unkey/pkg/tui/widget"

// Muted cool palette (256-color indices): steel blue is the primary accent,
// teal reads as healthy, amber/rose carry warnings and errors, dim gray is
// secondary. The reusable widget kit renders through this theme.
var theme = widget.NewTheme(widget.Palette{
	Border:    237,
	Title:     110, // muted steel blue (primary accent)
	Text:      250,
	Dim:       242,
	Selected:  238, // muted neutral selection bar
	SelText:   253,
	OK:        73,  // muted teal: up / running / healthy
	Mid:       180, // soft peach: mid / partial / scheduled
	Warn:      215, // soft orange: warnings
	Err:       174, // muted rose: errors / low
	Key:       109, // soft cyan: key hints
	Accent:    110, // steel blue accent (tab underline, markers)
	TabActive: 253,
})
