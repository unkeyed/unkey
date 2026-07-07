package ui

import "github.com/unkeyed/unkey/pkg/tui/app"

// Pane is one tab's content. Implement this and register in registry.go to add a tab.
type Pane interface {
	SetSize(w, h int)
	InitCmd() app.Cmd
	Update(msg app.Msg) app.Cmd
	Dispatch(msg app.Msg) (handled bool, cmd app.Cmd)
	View() string
	KeyHints() []string
	Status() string
	// DetailView renders the right-hand panel for the highlighted row at the
	// given content width, or "" if this pane has no detail panel. Panes that
	// return non-empty must be listed in detailTabs so the layout reserves room.
	DetailView(width int) string
	// Summary is a short, stable descriptor for the breadcrumb line (e.g.
	// "4 workspaces"). Empty means no summary.
	Summary() string
}
