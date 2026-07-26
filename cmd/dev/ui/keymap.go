package ui

import "github.com/unkeyed/unkey/pkg/tui/app"

// globalBind is one app-level key binding: the Binding drives both dispatch and
// the hint bar, do performs the action (mutating m, returning any command), and
// enabled optionally gates it. A disabled bind is neither shown nor matched, so
// its key falls through to the active pane (e.g. d on the Stripe tab, where the
// pane binds d to delete a clock). A nil do is a hint-only entry.
type globalBind struct {
	bind    app.Binding
	do      func(m *appModel, k app.KeyMsg) app.Cmd
	enabled func(m *appModel) bool
}

// globalBinds is the single source of truth for app-level keys and their hints.
// Order here is the order shown in the hint bar.
func globalBinds() []globalBind {
	switchTab := func(next func([]Tab, tabID) tabID) func(*appModel, app.KeyMsg) app.Cmd {
		return func(m *appModel, _ app.KeyMsg) app.Cmd {
			m.active = next(m.tabs, m.active)
			m.message = m.panes[m.active].Status()
			return nil
		}
	}
	return []globalBind{
		{bind: app.Binding{Keys: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}, Display: "1-5", Help: "jump"},
			do: func(m *appModel, k app.KeyMsg) app.Cmd {
				if id, ok := tabByHotkey(m.tabs, k.String()); ok {
					m.active = id
					m.message = m.panes[m.active].Status()
				}
				return nil
			}},
		{bind: app.Binding{Keys: []string{"]", "tab"}, Display: "tab", Help: "next"}, do: switchTab(nextTabID)},
		// prev has no hint (next covers the tab-cycling affordance).
		{bind: app.Binding{Keys: []string{"[", "shift+tab"}, Display: "", Help: ""}, do: switchTab(prevTabID)},
		// nav is owned by the panes; this is a hint-only entry.
		{bind: app.Binding{Keys: nil, Display: "↑/↓", Help: "nav"}},
		{bind: app.Binding{Keys: []string{"t"}, Display: "", Help: "tilt"}, do: func(m *appModel, _ app.KeyMsg) app.Cmd {
			openTilt()
			m.message = "Opened Tilt UI"
			return nil
		}},
		{bind: app.Binding{Keys: []string{"d"}, Display: "", Help: "dash"},
			enabled: func(m *appModel) bool { return m.active != tabStripe },
			do: func(m *appModel, _ app.KeyMsg) app.Cmd {
				openURL("http://localhost:3000")
				m.message = "Opened dashboard"
				return nil
			}},
		{bind: app.Binding{Keys: []string{"?"}, Display: "", Help: "help"}, do: func(m *appModel, _ app.KeyMsg) app.Cmd {
			m.helpOpen = !m.helpOpen
			return nil
		}},
		{bind: app.Binding{Keys: []string{"q", "ctrl+c"}, Display: "q", Help: "quit"}, do: func(m *appModel, _ app.KeyMsg) app.Cmd {
			m.quitting = true
			return app.Quit
		}},
	}
}

// paneKey is a pane-level binding: a Binding for dispatch + hints, and an
// action. A nil do is a hint-only entry (shown, never matched) for keys handled
// elsewhere (e.g. arrow navigation).
type paneKey struct {
	bind app.Binding
	do   func() app.Cmd
}

// keyList lets a pane drive its normal-mode dispatch and its hint bar from one
// declaration, so the two can never drift.
type keyList []paneKey

func (kl keyList) handle(msg app.KeyMsg) (app.Cmd, bool) {
	for _, k := range kl {
		if k.do == nil {
			continue
		}
		if k.bind.Matches(msg) {
			return k.do(), true
		}
	}
	return nil, false
}

func (kl keyList) hints() []string {
	h := make([]string, 0, len(kl))
	for _, k := range kl {
		if k.bind.Help == "" {
			continue
		}
		h = append(h, renderKeyHint(k.bind.DisplayKey(), k.bind.Help))
	}
	return h
}

// handleGlobalKey dispatches an app-level key, returning whether it was handled.
func (m *appModel) handleGlobalKey(msg app.KeyMsg) (app.Cmd, bool) {
	for _, g := range globalBinds() {
		if g.do == nil {
			continue // hint-only
		}
		if g.enabled != nil && !g.enabled(m) {
			continue // falls through to the active pane
		}
		if g.bind.Matches(msg) {
			return g.do(m, msg), true
		}
	}
	return nil, false
}

// globalKeyHints renders the hint bar from the same bindings, so keys and their
// help can never drift apart.
func (m appModel) globalKeyHints() []string {
	var hints []string
	for _, g := range globalBinds() {
		if g.bind.Help == "" {
			continue
		}
		if g.enabled != nil && !g.enabled(&m) {
			continue
		}
		hints = append(hints, renderKeyHint(g.bind.DisplayKey(), g.bind.Help))
	}
	return hints
}
