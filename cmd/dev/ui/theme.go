package ui

import (
	"github.com/unkeyed/unkey/pkg/tui/app"
	"github.com/unkeyed/unkey/pkg/tui/widget"
)

// This file binds the reusable pkg/tui/widget kit to the unkey dev tool: the
// widget types/functions keep their old local names so the pane code is
// unchanged, all forwarding to the package `theme`.

// Widget types.
type (
	tableColumn = widget.Column
	menuItem    = widget.MenuItem
	confirmLine = widget.ConfirmLine
	confirmNav  = widget.ConfirmNav
	detailField = widget.Field
)

const (
	detailPlain   = widget.DetailPlain
	detailOK      = widget.DetailOK
	detailMidTone = widget.DetailMid
	detailBad     = widget.DetailBad
	detailMuted   = widget.DetailMuted
)

// Palette styles used by pane code, bound to the unkey theme. (Styles used only
// inside the widget kit live in the Theme, not here.)
var (
	dimStyle         = theme.Dim
	okStyle          = theme.OK
	warnStyle        = theme.Warn
	errStyle         = theme.Err
	headerStyle      = theme.Header
	normalRowStyle   = theme.Normal
	selectedRowStyle = theme.SelectedRow
	tableHeaderStyle = theme.TableHeader
)

// Rendering widgets.
func renderKeyHint(key, label string) string { return theme.KeyHint(key, label) }
func sectionTitle(s string) string           { return theme.SectionTitle(s) }
func renderBodyFrame(w, h int, body string) string {
	return theme.Box(w, h, body)
}
func renderTable(w int, cols []tableColumn, rows [][]string, cursor int) string {
	return theme.Table(w, cols, rows, cursor)
}
func renderMenuList(w int, title string, items []menuItem, cursor int) string {
	return theme.MenuList(w, title, items, cursor)
}
func renderConfirmScreen(w int, title, subtitle string, lines []confirmLine, cursor int) string {
	return theme.ConfirmScreen(w, title, subtitle, lines, cursor)
}
func renderDetail(w int, title, subtitle string, fields []detailField) string {
	return theme.Detail(w, title, subtitle, fields)
}
func renderStatusLine(w int, msg string) string  { return theme.StatusLine(w, msg) }
func renderKeyBar(w int, hints ...string) string { return theme.KeyBar(w, hints...) }
func orDash(s string) string                     { return widget.OrDash(s) }
func renderContextLine(w int, crumb, summary string) string {
	return theme.ContextLine(w, crumb, summary)
}
func renderHeader(w int, right string) string { return theme.HeaderLine(w, "unkey dev", right) }
func confirmKeyHints(lines []confirmLine, cursor int) []string {
	return theme.ConfirmKeyHints(lines, cursor)
}

// renderTabBar formats the tab labels ("1 Stack", …) and delegates to the kit.
func renderTabBar(tabs []Tab, active tabID, width int) (line, underline string) {
	labels := make([]string, len(tabs))
	for i, t := range tabs {
		labels[i] = hotkeyForIndex(i) + " " + t.Label
	}
	return theme.TabBar(width, labels, tabIndex(tabs, active))
}

// Pure text/layout helpers.
func truncate(s string, n int) string          { return widget.Truncate(s, n) }
func padPlain(s string, w int) string          { return widget.PadPlain(s, w) }
func padToWidth(s string, w int) string        { return widget.Pad(s, w) }
func barWidth(w, contentW int) int             { return widget.BarWidth(w, contentW) }
func effectiveRowWidth(w int) int              { return widget.EffectiveRowWidth(w) }
func splitWidths(w int) (int, int, bool)       { return widget.SplitWidths(w) }
func joinHorizontal(left, right string) string { return widget.JoinHorizontal(left, right) }
func navigateConfirm(key app.KeyMsg, cursor int, lines []confirmLine) confirmNav {
	return widget.NavigateConfirm(key, cursor, lines)
}
func handleConfirmEnter(nav confirmNav, lines []confirmLine) (bool, bool) {
	return widget.HandleConfirmEnter(nav, lines)
}
func confirmFirstAdjustable(lines []confirmLine) int { return widget.ConfirmFirstAdjustable(lines) }

// Key predicates and cursor movement.
func keyUp(k app.KeyMsg) bool             { return k.IsUp() }
func keyDown(k app.KeyMsg) bool           { return k.IsDown() }
func keyEnter(k app.KeyMsg) bool          { return k.IsEnter() }
func keyEsc(k app.KeyMsg) bool            { return k.IsEsc() }
func keyLeft(k app.KeyMsg) bool           { return k.IsLeft() }
func keyRight(k app.KeyMsg) bool          { return k.IsRight() }
func moveCursor(cur, delta, maxN int) int { return app.MoveCursor(cur, delta, maxN) }

// Spinner tick helpers.
func withSpinnerTick(sp app.Spinner, cmd app.Cmd) app.Cmd { return app.WithSpinnerTick(sp, cmd) }
func handleSpinnerTick(sp app.Spinner, msg app.Msg) (app.Spinner, app.Cmd, bool) {
	return app.HandleSpinnerTick(sp, msg)
}
