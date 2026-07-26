package ui

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type appModel struct {
	tabs        []Tab
	panes       map[tabID]Pane
	active      tabID
	width       int
	height      int
	doctor      doctorReport
	doctorReady bool
	doctorBusy  bool
	message     string
	helpOpen    bool
	quitting    bool
}

func newAppModel() appModel {
	return appModel{ //nolint:exhaustruct
		tabs:   workflowTabs,
		panes:  newPaneMap(),
		active: tabStack,
		width:  80,
		height: 24,
		// Init launches the first doctor run; probes happen off the UI loop
		// so a hung docker daemon cannot freeze startup.
		doctorBusy: true,
	}
}

func (m appModel) Init() app.Cmd {
	cmds := []app.Cmd{app.EnterAltScreen, runDoctorCmd(), refreshTick()}
	for _, t := range m.tabs {
		if c := m.panes[t.ID].InitCmd(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return app.Batch(cmds...)
}

func (m appModel) Update(msg app.Msg) (app.Model, app.Cmd) {
	switch msg := msg.(type) {
	case app.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h := m.contentHeight()
		listOuter, _, split := splitWidths(msg.Width)
		for _, t := range m.tabs {
			w := msg.Width
			// Detail tabs render their list beside a fixed detail box when the
			// terminal is wide, so they lay out at the narrower list width.
			if split && isDetailTab(t.ID) {
				w = listOuter
			}
			m.panes[t.ID].SetSize(w, h)
		}
		return m, nil
	case doctorMsg:
		m.doctor = msg.report
		m.doctorReady = true
		m.doctorBusy = false
		return m, nil
	case refreshTickMsg:
		cmds := []app.Cmd{refreshTick(), checkPortsCmd(), loadProcsCmd()}
		if !m.doctorBusy {
			m.doctorBusy = true
			cmds = append(cmds, runDoctorCmd())
		}
		return m, app.Batch(cmds...)
	// Background refresh results are routed directly so they never clobber
	// m.message the way dispatchAsync does for user-triggered pane results.
	case portsMsg:
		_, cmd := m.panes[tabStack].Dispatch(msg)
		return m, cmd
	case procsMsg, procLogMsg:
		_, cmd := m.panes[tabProcs].Dispatch(msg)
		return m, cmd
	case app.KeyMsg:
		if m.quitting {
			return m, app.Quit
		}
		if m.helpOpen {
			if keyEsc(msg) || keyEnter(msg) || msg.String() == "?" {
				m.helpOpen = false
			}
			return m, nil
		}
		// App-level keys are declared once in globalBinds (dispatch + hints).
		if cmd, handled := m.handleGlobalKey(msg); handled {
			return m, cmd
		}
	}

	if handled, cmd := m.dispatchAsync(msg); handled {
		return m, cmd
	}

	cmd := m.panes[m.active].Update(msg)
	if st := m.panes[m.active].Status(); st != "" {
		m.message = st
	}
	return m, cmd
}

// Pointer receiver: the message write below must survive into the model the
// caller returns; with a value receiver it silently mutated a copy.
func (m *appModel) dispatchAsync(msg app.Msg) (bool, app.Cmd) {
	for _, t := range m.tabs {
		if handled, cmd := m.panes[t.ID].Dispatch(msg); handled {
			if t.ID == m.active {
				m.message = m.panes[t.ID].Status()
			}
			return true, cmd
		}
	}
	return false, nil
}

// chromeRows is every line rendered outside the body frame: header, tab line,
// tab underline, context line, key bar, and the always-present status line. The
// body frame gets the rest of the terminal so the app fills the alt screen.
const chromeRows = 6

// detailTabs render a right-hand detail panel and lay their list out narrower
// when the terminal is wide. Keep in sync with the panes that return a
// non-empty DetailView.
func isDetailTab(id tabID) bool {
	return id == tabSeed || id == tabStripe || id == tabProcs
}

func (m appModel) contentHeight() int {
	h := m.height - chromeRows
	if h < 8 {
		return 8
	}
	return h
}

func (m appModel) View() string {
	if m.quitting {
		return ""
	}
	// Help takes the whole body; a split detail box would only distract.
	if m.helpOpen {
		return m.renderScreen(renderBodyFrame(m.width, m.contentHeight(), m.helpView()))
	}
	return m.renderScreen(m.renderBodyRegion())
}

// renderBodyRegion is the framed content area: a list + detail split on detail
// tabs when the terminal is wide, otherwise a single full-width frame.
func (m appModel) renderBodyRegion() string {
	h := m.contentHeight()
	active := m.panes[m.active]
	listOuter, detOuter, split := splitWidths(m.width)
	if split && isDetailTab(m.active) {
		left := renderBodyFrame(listOuter, h, active.View())
		right := renderBodyFrame(detOuter, h, active.DetailView(detOuter))
		return joinHorizontal(left, right)
	}
	return renderBodyFrame(m.width, h, active.View())
}

func (m appModel) renderScreen(frame string) string {
	header := renderHeader(m.width, m.doctorLine())
	tabLine, tabUnderline := renderTabBar(m.tabs, m.active, m.width)
	context := renderContextLine(m.width, m.breadcrumb(), m.panes[m.active].Summary())

	hints := append(m.globalKeyHints(), m.panes[m.active].KeyHints()...)
	keyBar := renderKeyBar(m.width, hints...)

	// Every row is rendered unconditionally so the layout never shifts when
	// a status message appears or clears. Keep in sync with chromeRows.
	status := renderStatusLine(m.width, m.message)
	return strings.Join([]string{header, tabLine, tabUnderline, context, frame, keyBar, status}, "\n")
}

func (m appModel) breadcrumb() string {
	for _, t := range m.tabs {
		if t.ID == m.active {
			return strings.ToLower(t.Label)
		}
	}
	return ""
}

func (m appModel) doctorLine() string {
	if !m.doctorReady {
		return dimStyle.Render("checking environment...")
	}
	return doctorSummary(m.doctor)
}

func (m appModel) helpView() string {
	lines := []string{
		headerStyle.Render("unkey dev"),
		"",
		"Workflow: Stack → Seed → Stripe → GitHub · Procs",
		"",
		"Stack starts or tears down local services. Seed lists workspaces and runs seeds.",
		"Stripe shows test clocks inline. GitHub runs webhook dev helpers.",
		"Procs lists processes started here: tail logs, stop, or kill them.",
		"They keep running after you quit; this TUI reattaches on the next launch.",
		"",
		"t opens the Tilt UI, d opens the dashboard (except on the Stripe tab).",
		"",
		"Env vars drive configuration (UNKEY_DATABASE_PRIMARY, dev/.env.stripe, etc.).",
		"No forms. Override in your shell or dotenv files before launching.",
		"",
		headerStyle.Render("Agents and CI"),
		"",
		"This TUI is for humans. Agents and scripts should use the headless CLI:",
		"  unkey dev seed local",
		"  unkey dev stripe clock advance",
		"  mise run dev",
	}
	if m.doctorReady && len(m.doctor.Messages) > 0 {
		lines = append(lines, "", headerStyle.Render("Doctor"), "")
		for _, msg := range m.doctor.Messages {
			lines = append(lines, warnStyle.Render("- "+msg))
		}
	}
	return stringsJoinLines(lines...)
}

func stringsJoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}
