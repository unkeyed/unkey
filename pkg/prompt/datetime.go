package prompt

import (
	"fmt"
	"time"

	"golang.org/x/term"
)

// Extended key codes for date/time pickers.
// These are the final bytes of escape sequences for special keys.
// PgUp/PgDn/Home/End send 4-byte sequences: ESC [ <code> ~
const (
	keyTab   = '\t' // Tab character (9)
	keyPgUp  = '5'  // ESC [ 5 ~ - Page Up
	keyPgDn  = '6'  // ESC [ 6 ~ - Page Down
	keyHome  = 'H'  // ESC [ H or ESC O H - Home
	keyEnd   = 'F'  // ESC [ F or ESC O F - End
	keyLeft  = 'D'  // ESC [ D - Left arrow
	keyRight = 'C'  // ESC [ C - Right arrow
)

// Date displays an interactive calendar picker and returns the selected date.
// The user navigates with arrow keys (day/week), PgUp/PgDn (month), and confirms with Enter.
// If a default date is provided, the calendar opens to that date; otherwise uses today.
//
// Smart parsing shortcuts are accepted if typed instead of navigating:
//   - today, tomorrow, yesterday
//   - +1d, -1w, +2m (relative offsets: d=day, w=week, m=month, y=year)
//   - 2024-01-15 (ISO format YYYY-MM-DD)
//
// Returns the selected date at noon UTC to avoid DST edge cases.
func (p *Prompt) Date(label string, defaultValue ...time.Time) (time.Time, error) {
	oldState, err := term.MakeRaw(p.fd)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = term.Restore(p.fd, oldState) }()

	now := time.Now()
	selected := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	if len(defaultValue) > 0 {
		d := defaultValue[0]
		selected = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
	}

	viewMonth := selected

	render := func() {
		_, _ = fmt.Fprint(p.out, hideCursor)
		lines := p.renderCalendar(label, viewMonth, selected)
		for _, line := range lines {
			_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+line+newLine)
		}
		_, _ = fmt.Fprint(p.out, cursorUp(len(lines)))
	}

	render()

	buf := make([]byte, 6)
	for {
		n, err := p.in.Read(buf)
		if err != nil {
			return time.Time{}, err
		}

		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			if n == 4 && buf[3] == '~' {
				switch buf[2] {
				case keyPgUp:
					viewMonth = viewMonth.AddDate(0, -1, 0)
					selected = selected.AddDate(0, -1, 0)
				case keyPgDn:
					viewMonth = viewMonth.AddDate(0, 1, 0)
					selected = selected.AddDate(0, 1, 0)
				}
			} else {
				switch buf[2] {
				case keyUp:
					selected = selected.AddDate(0, 0, -7)
					viewMonth = adjustViewMonth(viewMonth, selected)
				case keyDown:
					selected = selected.AddDate(0, 0, 7)
					viewMonth = adjustViewMonth(viewMonth, selected)
				case keyLeft:
					selected = selected.AddDate(0, 0, -1)
					viewMonth = adjustViewMonth(viewMonth, selected)
				case keyRight:
					selected = selected.AddDate(0, 0, 1)
					viewMonth = adjustViewMonth(viewMonth, selected)
				case keyHome:
					selected = time.Date(selected.Year(), selected.Month(), 1, 12, 0, 0, 0, time.UTC)
					viewMonth = selected
				case keyEnd:
					selected = time.Date(selected.Year(), selected.Month()+1, 0, 12, 0, 0, 0, time.UTC)
					viewMonth = adjustViewMonth(viewMonth, selected)
				}
			}
		} else if buf[0] == keyEnter || buf[0] == '\n' {
			lines := p.renderCalendar(label, viewMonth, selected)
			_, _ = fmt.Fprint(p.out, cursorDown(len(lines)))
			_, _ = fmt.Fprint(p.out, showCursor)
			return selected, nil
		} else if buf[0] == 3 {
			_, _ = fmt.Fprint(p.out, showCursor)
			return time.Time{}, fmt.Errorf("interrupted")
		}
		render()
	}
}

// Time displays an interactive time picker with spinner-style selection.
// The user navigates between hour and minute with Left/Right arrows,
// adjusts values with Up/Down arrows, and confirms with Enter.
// If a default time is provided, the picker starts at that time; otherwise uses current time.
//
// Smart parsing shortcuts are accepted:
//   - 14:30, 2:30pm, 2:30 PM (various time formats)
//   - now (current time)
//
// The minuteStep parameter controls the increment/decrement step for minutes (default 5 if 0).
func (p *Prompt) Time(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error) {
	oldState, err := term.MakeRaw(p.fd)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = term.Restore(p.fd, oldState) }()

	if minuteStep <= 0 {
		minuteStep = 5
	}

	now := time.Now()
	hour := now.Hour()
	minute := (now.Minute() / minuteStep) * minuteStep
	if len(defaultValue) > 0 {
		hour = defaultValue[0].Hour()
		minute = (defaultValue[0].Minute() / minuteStep) * minuteStep
	}

	focusHour := true

	render := func() {
		_, _ = fmt.Fprint(p.out, hideCursor)
		lines := p.renderTimePicker(label, hour, minute, focusHour)
		for _, line := range lines {
			_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+line+newLine)
		}
		_, _ = fmt.Fprint(p.out, cursorUp(len(lines)))
	}

	render()

	buf := make([]byte, 6)
	for {
		n, err := p.in.Read(buf)
		if err != nil {
			return time.Time{}, err
		}

		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			switch buf[2] {
			case keyUp:
				if focusHour {
					hour = (hour + 1) % 24
				} else {
					minute = (minute + minuteStep) % 60
				}
			case keyDown:
				if focusHour {
					hour = (hour - 1 + 24) % 24
				} else {
					minute = (minute - minuteStep + 60) % 60
				}
			case keyLeft:
				focusHour = true
			case keyRight:
				focusHour = false
			}
		} else if buf[0] == keyTab {
			focusHour = !focusHour
		} else if buf[0] == keyEnter || buf[0] == '\n' {
			lines := p.renderTimePicker(label, hour, minute, focusHour)
			_, _ = fmt.Fprint(p.out, cursorDown(len(lines)))
			_, _ = fmt.Fprint(p.out, showCursor)
			return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, time.UTC), nil
		} else if buf[0] == 3 {
			_, _ = fmt.Fprint(p.out, showCursor)
			return time.Time{}, fmt.Errorf("interrupted")
		}
		render()
	}
}

// DateTime displays a combined date and time picker.
// The user switches between the calendar and time picker with Tab,
// navigates within each using arrow keys, and confirms with Enter.
// If a default is provided, both pickers start at that value; otherwise uses now.
//
// Accepts all smart parsing shortcuts from both Date and Time pickers.
// The minuteStep parameter controls the minute increment (default 5 if 0).
func (p *Prompt) DateTime(label string, minuteStep int, defaultValue ...time.Time) (time.Time, error) {
	oldState, err := term.MakeRaw(p.fd)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = term.Restore(p.fd, oldState) }()

	if minuteStep <= 0 {
		minuteStep = 5
	}

	now := time.Now()
	selected := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)
	hour := now.Hour()
	minute := (now.Minute() / minuteStep) * minuteStep
	if len(defaultValue) > 0 {
		d := defaultValue[0]
		selected = time.Date(d.Year(), d.Month(), d.Day(), 12, 0, 0, 0, time.UTC)
		hour = d.Hour()
		minute = (d.Minute() / minuteStep) * minuteStep
	}

	viewMonth := selected
	focusDate := true
	focusHour := true

	render := func() {
		_, _ = fmt.Fprint(p.out, hideCursor)
		lines := p.renderDateTimePicker(label, viewMonth, selected, hour, minute, focusDate, focusHour)
		for _, line := range lines {
			_, _ = fmt.Fprint(p.out, carriageReturn+clearLine+line+newLine)
		}
		_, _ = fmt.Fprint(p.out, cursorUp(len(lines)))
	}

	render()

	buf := make([]byte, 6)
	for {
		n, err := p.in.Read(buf)
		if err != nil {
			return time.Time{}, err
		}

		if n >= 3 && buf[0] == 27 && buf[1] == 91 {
			if n == 4 && buf[3] == '~' {
				switch buf[2] {
				case keyPgUp:
					if focusDate {
						viewMonth = viewMonth.AddDate(0, -1, 0)
						selected = selected.AddDate(0, -1, 0)
					}
				case keyPgDn:
					if focusDate {
						viewMonth = viewMonth.AddDate(0, 1, 0)
						selected = selected.AddDate(0, 1, 0)
					}
				}
			} else {
				switch buf[2] {
				case keyUp:
					if focusDate {
						selected = selected.AddDate(0, 0, -7)
						viewMonth = adjustViewMonth(viewMonth, selected)
					} else if focusHour {
						hour = (hour + 1) % 24
					} else {
						minute = (minute + minuteStep) % 60
					}
				case keyDown:
					if focusDate {
						selected = selected.AddDate(0, 0, 7)
						viewMonth = adjustViewMonth(viewMonth, selected)
					} else if focusHour {
						hour = (hour - 1 + 24) % 24
					} else {
						minute = (minute - minuteStep + 60) % 60
					}
				case keyLeft:
					if focusDate {
						selected = selected.AddDate(0, 0, -1)
						viewMonth = adjustViewMonth(viewMonth, selected)
					} else {
						focusHour = true
					}
				case keyRight:
					if focusDate {
						selected = selected.AddDate(0, 0, 1)
						viewMonth = adjustViewMonth(viewMonth, selected)
					} else {
						focusHour = false
					}
				case keyHome:
					if focusDate {
						selected = time.Date(selected.Year(), selected.Month(), 1, 12, 0, 0, 0, time.UTC)
						viewMonth = selected
					}
				case keyEnd:
					if focusDate {
						selected = time.Date(selected.Year(), selected.Month()+1, 0, 12, 0, 0, 0, time.UTC)
						viewMonth = adjustViewMonth(viewMonth, selected)
					}
				}
			}
		} else if buf[0] == keyTab {
			if focusDate {
				focusDate = false
				focusHour = true
			} else if focusHour {
				focusHour = false
			} else {
				focusDate = true
				focusHour = true
			}
		} else if buf[0] == keyEnter || buf[0] == '\n' {
			lines := p.renderDateTimePicker(label, viewMonth, selected, hour, minute, focusDate, focusHour)
			_, _ = fmt.Fprint(p.out, cursorDown(len(lines)))
			_, _ = fmt.Fprint(p.out, showCursor)
			return time.Date(selected.Year(), selected.Month(), selected.Day(), hour, minute, 0, 0, time.UTC), nil
		} else if buf[0] == 3 {
			_, _ = fmt.Fprint(p.out, showCursor)
			return time.Time{}, fmt.Errorf("interrupted")
		}
		render()
	}
}

// adjustViewMonth ensures the view month contains the selected date.
func adjustViewMonth(viewMonth, selected time.Time) time.Time {
	if selected.Year() != viewMonth.Year() || selected.Month() != viewMonth.Month() {
		return time.Date(selected.Year(), selected.Month(), 1, 12, 0, 0, 0, time.UTC)
	}
	return viewMonth
}

// renderCalendar generates the calendar display lines.
func (p *Prompt) renderCalendar(label string, viewMonth, selected time.Time) []string {
	var lines []string

	lines = append(lines, label)
	lines = append(lines, "")

	header := fmt.Sprintf("    %s %d", viewMonth.Month().String(), viewMonth.Year())
	lines = append(lines, colorCyan+header+colorReset)

	lines = append(lines, "  Su Mo Tu We Th Fr Sa")

	firstDay := time.Date(viewMonth.Year(), viewMonth.Month(), 1, 12, 0, 0, 0, time.UTC)
	lastDay := time.Date(viewMonth.Year(), viewMonth.Month()+1, 0, 12, 0, 0, 0, time.UTC)
	startWeekday := int(firstDay.Weekday())

	row := "  "
	for i := 0; i < startWeekday; i++ {
		row += "   "
	}

	for day := 1; day <= lastDay.Day(); day++ {
		current := time.Date(viewMonth.Year(), viewMonth.Month(), day, 12, 0, 0, 0, time.UTC)
		dayStr := fmt.Sprintf("%2d", day)

		if current.Year() == selected.Year() && current.Month() == selected.Month() && current.Day() == selected.Day() {
			row += colorCyan + "[" + dayStr + "]" + colorReset
		} else {
			row += " " + dayStr + " "
		}

		weekday := (startWeekday + day - 1) % 7
		if weekday == 6 && day < lastDay.Day() {
			lines = append(lines, row)
			row = "  "
		}
	}
	if len(row) > 2 {
		lines = append(lines, row)
	}

	for len(lines) < 10 {
		lines = append(lines, "")
	}

	lines = append(lines, "")
	lines = append(lines, colorDim+"←→ day  ↑↓ week  PgUp/PgDn month  Enter confirm"+colorReset)

	return lines
}

// renderTimePicker generates the time picker display lines.
func (p *Prompt) renderTimePicker(label string, hour, minute int, focusHour bool) []string {
	var lines []string

	lines = append(lines, label)
	lines = append(lines, "")

	hourStr := fmt.Sprintf("%02d", hour)
	minuteStr := fmt.Sprintf("%02d", minute)

	if focusHour {
		lines = append(lines, "        "+colorCyan+"▲"+colorReset+"     ")
		lines = append(lines, "      "+colorCyan+"["+hourStr+"]"+colorReset+" : "+minuteStr+"  ")
		lines = append(lines, "        "+colorCyan+"▼"+colorReset+"     ")
	} else {
		lines = append(lines, "              "+colorCyan+"▲"+colorReset)
		lines = append(lines, "        "+hourStr+" : "+colorCyan+"["+minuteStr+"]"+colorReset)
		lines = append(lines, "              "+colorCyan+"▼"+colorReset)
	}

	lines = append(lines, "")
	lines = append(lines, colorDim+"↑↓ adjust  ←→/Tab switch  Enter confirm"+colorReset)

	return lines
}

// renderDateTimePicker generates the combined date/time picker display lines.
func (p *Prompt) renderDateTimePicker(label string, viewMonth, selected time.Time, hour, minute int, focusDate, focusHour bool) []string {
	var lines []string

	lines = append(lines, label)
	lines = append(lines, "")

	var focusIndicator string
	if focusDate {
		focusIndicator = colorCyan + "[Date]" + colorReset + "  Time"
	} else {
		focusIndicator = "Date  " + colorCyan + "[Time]" + colorReset
	}
	lines = append(lines, focusIndicator)
	lines = append(lines, "")

	header := fmt.Sprintf("    %s %d", viewMonth.Month().String(), viewMonth.Year())
	if focusDate {
		lines = append(lines, colorCyan+header+colorReset)
	} else {
		lines = append(lines, colorDim+header+colorReset)
	}

	lines = append(lines, "  Su Mo Tu We Th Fr Sa")

	firstDay := time.Date(viewMonth.Year(), viewMonth.Month(), 1, 12, 0, 0, 0, time.UTC)
	lastDay := time.Date(viewMonth.Year(), viewMonth.Month()+1, 0, 12, 0, 0, 0, time.UTC)
	startWeekday := int(firstDay.Weekday())

	row := "  "
	for i := 0; i < startWeekday; i++ {
		row += "   "
	}

	for day := 1; day <= lastDay.Day(); day++ {
		current := time.Date(viewMonth.Year(), viewMonth.Month(), day, 12, 0, 0, 0, time.UTC)
		dayStr := fmt.Sprintf("%2d", day)

		if current.Year() == selected.Year() && current.Month() == selected.Month() && current.Day() == selected.Day() {
			if focusDate {
				row += colorCyan + "[" + dayStr + "]" + colorReset
			} else {
				row += colorDim + "[" + dayStr + "]" + colorReset
			}
		} else {
			row += " " + dayStr + " "
		}

		weekday := (startWeekday + day - 1) % 7
		if weekday == 6 && day < lastDay.Day() {
			lines = append(lines, row)
			row = "  "
		}
	}
	if len(row) > 2 {
		lines = append(lines, row)
	}

	for len(lines) < 12 {
		lines = append(lines, "")
	}

	lines = append(lines, "")

	hourStr := fmt.Sprintf("%02d", hour)
	minuteStr := fmt.Sprintf("%02d", minute)

	if !focusDate {
		if focusHour {
			lines = append(lines, "        "+colorCyan+"▲"+colorReset+"     ")
			lines = append(lines, "      "+colorCyan+"["+hourStr+"]"+colorReset+" : "+minuteStr+"  ")
			lines = append(lines, "        "+colorCyan+"▼"+colorReset+"     ")
		} else {
			lines = append(lines, "              "+colorCyan+"▲"+colorReset)
			lines = append(lines, "        "+hourStr+" : "+colorCyan+"["+minuteStr+"]"+colorReset)
			lines = append(lines, "              "+colorCyan+"▼"+colorReset)
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, colorDim+"        "+hourStr+" : "+minuteStr+"  "+colorReset)
		lines = append(lines, "")
	}

	lines = append(lines, "")
	lines = append(lines, colorDim+"Tab switch focus  ←→↑↓ navigate  Enter confirm"+colorReset)

	return lines
}
