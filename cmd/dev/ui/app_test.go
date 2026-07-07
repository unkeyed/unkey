package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

func keyRune(r rune) app.KeyMsg {
	return app.KeyMsg{Type: app.KeyRunes, Runes: []rune{r}, Alt: false, Paste: false}
}

func updated(t *testing.T, m app.Model, msg app.Msg) appModel {
	t.Helper()
	next, _ := m.Update(msg)
	app, ok := next.(appModel)
	require.True(t, ok)
	return app
}

func newTestApp(t *testing.T) appModel {
	t.Helper()
	m := newAppModel()
	return updated(t, m, app.WindowSizeMsg{Width: 100, Height: 28})
}

func TestHotkeysJumpToEveryTab(t *testing.T) {
	chdirTemp(t) // procs pane lists .cache/devui relative to cwd
	m := newTestApp(t)
	for i, tab := range workflowTabs {
		m = updated(t, m, keyRune(rune('1'+i)))
		require.Equal(t, tab.ID, m.active, "hotkey %d", i+1)
	}
}

func TestProcsTabRendersEmptyState(t *testing.T) {
	chdirTemp(t)
	m := newTestApp(t)
	m = updated(t, m, keyRune('5'))
	require.Equal(t, tabProcs, m.active)
	require.Contains(t, m.View(), "No managed processes")
}

func TestTabKeyCycles(t *testing.T) {
	chdirTemp(t)
	m := newTestApp(t)
	require.Equal(t, tabStack, m.active)
	m = updated(t, m, app.KeyMsg{Type: app.KeyTab}) //nolint:exhaustruct
	require.Equal(t, tabSeed, m.active)
}

func TestDashboardKeyYieldsToStripePane(t *testing.T) {
	chdirTemp(t)
	m := newTestApp(t)
	m = updated(t, m, keyRune('3'))
	require.Equal(t, tabStripe, m.active)
	// d must reach the stripe pane (delete clock), not open the dashboard.
	m = updated(t, m, keyRune('d'))
	require.NotEqual(t, "Opened dashboard", m.message)
}

func TestStackEnterConfirmsOnlyDestructiveTasks(t *testing.T) {
	chdirTemp(t)
	m := newTestApp(t)
	pane, ok := m.panes[tabStack].(*stackPane)
	require.True(t, ok)

	for i, task := range pane.tasks {
		require.Equal(t, task.confirm, task.task == "down" || task.task == "nuke",
			"task %d (%s): confirm flag", i, task.task)
	}

	// Move to "down" and press enter: a confirm screen must open instead of
	// the task starting.
	for pane.cursor < len(pane.tasks)-2 {
		m = updated(t, m, app.KeyMsg{Type: app.KeyDown}) //nolint:exhaustruct
	}
	require.Equal(t, "down", pane.tasks[pane.cursor].task)
	m = updated(t, m, app.KeyMsg{Type: app.KeyEnter}) //nolint:exhaustruct
	require.NotNil(t, pane.confirm)
	view := m.View()
	require.Contains(t, view, "Run stack task")
	require.Contains(t, view, "Tear down")
}

func TestScreenAlwaysFillsTerminalHeight(t *testing.T) {
	chdirTemp(t)
	m := newTestApp(t)
	view := m.View()
	require.Equal(t, 28, strings.Count(view, "\n")+1, "screen must fill the terminal exactly")

	// A status message must not change the row count.
	m.message = "something happened"
	view = m.View()
	require.Equal(t, 28, strings.Count(view, "\n")+1)
}
