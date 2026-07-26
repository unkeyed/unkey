package widget

// Palette is the set of 256-color indices a Theme draws from. Callers name
// their own colors here.
type Palette struct {
	Border    int
	Title     int
	Text      int
	Dim       int
	Selected  int // selection-bar background
	SelText   int // selection-bar foreground
	OK        int
	Mid       int
	Warn      int
	Err       int
	Key       int
	Accent    int
	TabActive int
}

// Theme carries a resolved palette as ready-to-use styles plus the widget
// render methods. Construct it with NewTheme; the exported Style fields let
// callers style their own text in the same palette.
type Theme struct {
	color bool

	Border      Style
	Dim         Style
	Text        Style
	OK          Style
	Mid         Style
	Warn        Style
	Err         Style
	Accent      Style
	Title       Style
	Header      Style
	SelectedRow Style
	DetailLabel Style
	DetailValue Style
	TabActive   Style
	TabInactive Style
	TableHeader Style
	TableRule   Style
	Normal      Style
	Key         Style
}

func NewTheme(p Palette) Theme {
	on := DetectColor()
	fg := func(code int) Style { return Style{fg: code, bg: -1, bold: false, on: on} }
	boldFg := func(code int) Style { return Style{fg: code, bg: -1, bold: true, on: on} }
	return Theme{
		color:       on,
		Border:      fg(p.Border),
		Dim:         fg(p.Dim),
		Text:        fg(p.Text),
		OK:          fg(p.OK),
		Mid:         fg(p.Mid),
		Warn:        fg(p.Warn),
		Err:         fg(p.Err),
		Accent:      fg(p.Accent),
		Title:       boldFg(p.Title),
		Header:      boldFg(p.Text),
		SelectedRow: Style{fg: p.SelText, bg: p.Selected, bold: true, on: on},
		DetailLabel: fg(p.Dim),
		DetailValue: fg(p.Text),
		TabActive:   boldFg(p.TabActive),
		TabInactive: fg(p.Dim),
		TableHeader: boldFg(p.Dim),
		TableRule:   fg(p.Border),
		Normal:      fg(p.Text),
		Key:         fg(p.Key),
	}
}
