package ui

import (
	"fmt"
	"os"
	"strings"

	devseed "github.com/unkeyed/unkey/internal/devtools/seed"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

type seedPane struct {
	workspaces    []workspaceRow
	cursor        int
	width         int
	height        int
	loading       bool
	busy          bool
	busyText      string
	statusMessage string
	loadError     string
	activeWS      string
	lastRootKey   string
	checkpoint    *checkpointWizard
	seedConfirm   *seedActionConfirm
	// Inline tree state: which workspaces are expanded and their loaded APIs.
	// apis[id] == nil means "not loaded yet"; a non-nil empty slice means
	// "loaded, none found" so it is not re-queried.
	expanded   map[string]bool
	apis       map[string][]namedChoice
	apiLoading map[string]bool
	spinner    app.Spinner
}

func newSeedPane() *seedPane {
	s := app.NewSpinner()
	return &seedPane{ //nolint:exhaustruct
		spinner:    s,
		loading:    true,
		activeWS:   readActiveWorkspaceID(),
		expanded:   map[string]bool{},
		apis:       map[string][]namedChoice{},
		apiLoading: map[string]bool{},
		// dev/.env.seed is loaded into the process env at startup, so the
		// root key from the last seed survives TUI restarts.
		lastRootKey: os.Getenv("UNKEY_ROOT_KEY"),
	}
}

func (p *seedPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *seedPane) InitCmd() app.Cmd { return p.refreshCmd() }

func (p *seedPane) refreshCmd() app.Cmd {
	p.loading = true
	p.loadError = ""
	p.activeWS = readActiveWorkspaceID()
	// Drop stale expansion state so a refresh re-queries APIs.
	p.expanded = map[string]bool{}
	p.apis = map[string][]namedChoice{}
	p.apiLoading = map[string]bool{}
	return withSpinnerTick(p.spinner, loadWorkspacesCmd())
}

func (p *seedPane) Dispatch(msg app.Msg) (bool, app.Cmd) {
	switch msg := msg.(type) {
	case workspacesMsg:
		p.loading = false
		p.busy = false
		if msg.err != nil {
			p.loadError = msg.err.Error()
			p.statusMessage = msg.err.Error()
			return true, nil
		}
		p.loadError = ""
		p.workspaces = msg.rows
		p.cursor = moveCursor(p.cursor, 0, len(p.workspaces))
		p.statusMessage = fmt.Sprintf("%d workspaces", len(p.workspaces))
		return true, nil
	case seedExecMsg:
		p.busy = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
			if msg.output != "" {
				p.statusMessage += "\n" + strings.TrimSpace(msg.output)
			}
		} else if msg.output != "" {
			p.statusMessage = strings.TrimSpace(msg.output)
		} else {
			p.statusMessage = "Done"
		}
		return true, nil
	case apiChoicesMsg:
		p.apiLoading[msg.workspaceID] = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
			p.apis[msg.workspaceID] = []namedChoice{}
			return true, nil
		}
		choices := msg.choices
		if choices == nil {
			choices = []namedChoice{}
		}
		p.apis[msg.workspaceID] = choices
		p.clampCursor()
		return true, nil
	case checkpointChoicesMsg:
		if p.checkpoint == nil || p.checkpoint.step != msg.step {
			return true, nil
		}
		p.checkpoint.loading = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
			p.checkpoint = nil
			return true, nil
		}
		p.checkpoint.choices = msg.choices
		p.checkpoint.cursor = moveCursor(p.checkpoint.cursor, 0, len(msg.choices))
		return true, nil
	case seedLocalDoneMsg:
		p.busy = false
		if msg.err != nil {
			p.statusMessage = msg.err.Error()
		} else {
			p.lastRootKey = msg.result.RootKey
			p.statusMessage = fmt.Sprintf("Seeded %s · root %s · y to copy", msg.result.WorkspaceID, msg.result.RootKey)
		}
		return true, p.refreshCmd()
	case app.SpinnerTickMsg:
		if p.busy || p.loading || (p.checkpoint != nil && p.checkpoint.loading) || p.apiLoadingAny() {
			var cmd app.Cmd
			var handled bool
			p.spinner, cmd, handled = handleSpinnerTick(p.spinner, msg)
			if handled {
				return true, cmd
			}
		}
	}
	return false, nil
}

func (p *seedPane) Update(msg app.Msg) app.Cmd {
	key, ok := msg.(app.KeyMsg)
	if !ok || p.busy {
		return nil
	}
	if p.checkpoint != nil {
		return p.updateCheckpoint(key)
	}
	if p.seedConfirm != nil {
		return p.updateSeedConfirm(key)
	}
	switch {
	case keyUp(key):
		p.cursor = moveCursor(p.cursor, -1, len(p.visibleRows()))
		return nil
	case keyDown(key):
		p.cursor = moveCursor(p.cursor, 1, len(p.visibleRows()))
		return nil
	case keyRight(key):
		return p.toggleExpand()
	case keyLeft(key):
		p.collapseLeft()
		return nil
	}
	if cmd, ok := p.keys().handle(key); ok {
		return cmd
	}
	return nil
}

func (p *seedPane) keys() keyList {
	kl := keyList{
		// right/left drive expand/collapse in the nav block above; this is the
		// shared hint for them.
		{bind: app.Binding{Display: "→/←", Help: "expand"}},
		{bind: app.Binding{Keys: []string{"l"}, Help: "local"}, do: func() app.Cmd { return p.openSeedConfirm(seedActionLocal) }},
		{bind: app.Binding{Keys: []string{"f"}, Help: "frontline"}, do: func() app.Cmd { return p.openSeedConfirm(seedActionFrontline) }},
		{bind: app.Binding{Keys: []string{"v"}, Help: "verifications"}, do: func() app.Cmd { return p.startVerificationsFromCursor() }},
		{bind: app.Binding{Keys: []string{"c"}, Help: "checkpoints"}, do: func() app.Cmd { return p.startCheckpointWizard() }},
		{bind: app.Binding{Keys: []string{"r"}, Help: "refresh"}, do: func() app.Cmd { return p.refreshCmd() }},
		// Enter picks an API for verifications or toggles a workspace; no hint.
		{bind: app.Binding{Keys: []string{"enter"}}, do: func() app.Cmd { return p.activateRow() }},
	}
	if p.lastRootKey != "" {
		kl = append(kl, paneKey{bind: app.Binding{Keys: []string{"y"}, Help: "root key"}, do: func() app.Cmd {
			p.copyRootKey()
			return nil
		}})
	}
	return kl
}

// activateRow handles Enter: toggle a workspace, or run verifications on an API.
func (p *seedPane) activateRow() app.Cmd {
	row, ok := p.currentRow()
	if !ok {
		return nil
	}
	if row.kind == rowAPI {
		return p.startVerifications(row.wsID, row.api)
	}
	return p.toggleExpand()
}

// startVerificationsFromCursor runs verifications when an API row is selected,
// otherwise expands the workspace and points the user at its APIs.
func (p *seedPane) startVerificationsFromCursor() app.Cmd {
	row, ok := p.currentRow()
	if !ok {
		p.statusMessage = "select a workspace first"
		return nil
	}
	switch row.kind {
	case rowAPI:
		return p.startVerifications(row.wsID, row.api)
	case rowWorkspace:
		p.statusMessage = "pick an API below, then press v"
		if !p.expanded[row.ws.ID] {
			return p.toggleExpand()
		}
		return nil
	case rowAPIEmpty:
		p.statusMessage = "no APIs here yet; seed one with l"
		return nil
	case rowAPILoading:
		return nil
	}
	return nil
}

func (p *seedPane) copyRootKey() {
	if p.lastRootKey == "" {
		p.statusMessage = "no root key yet; run a local seed first"
		return
	}
	if err := copyToClipboard(p.lastRootKey); err != nil {
		p.statusMessage = err.Error()
		return
	}
	p.statusMessage = "Root key copied to clipboard"
}

func (p *seedPane) updateCheckpoint(key app.KeyMsg) app.Cmd {
	w := p.checkpoint
	if w == nil {
		return nil
	}
	if keyEsc(key) {
		if w.step == checkpointStepProject {
			p.checkpoint = nil
			return nil
		}
		p.checkpointBack()
		if p.checkpoint != nil && p.checkpoint.loading {
			return p.loadCheckpointChoices(p.checkpoint.step)
		}
		return nil
	}
	if w.loading {
		return nil
	}
	if w.step == checkpointStepConfirm {
		return p.updateCheckpointConfirm(key)
	}
	switch {
	case keyUp(key):
		w.cursor = moveCursor(w.cursor, -1, len(w.choices))
	case keyDown(key):
		w.cursor = moveCursor(w.cursor, 1, len(w.choices))
	case keyEnter(key):
		if len(w.choices) == 0 {
			return nil
		}
		ch := w.choices[w.cursor]
		switch w.step {
		case checkpointStepProject:
			w.projectID = ch.ID
			w.projectName = ch.Label
			w.step = checkpointStepApp
			w.cursor = 0
			w.loading = true
			return p.loadCheckpointChoices(checkpointStepApp)
		case checkpointStepApp:
			w.appID = ch.ID
			w.appName = ch.Label
			w.step = checkpointStepEnv
			w.cursor = 0
			w.loading = true
			return p.loadCheckpointChoices(checkpointStepEnv)
		case checkpointStepEnv:
			w.envID = ch.ID
			w.envName = ch.Label
			w.step = checkpointStepConfirm
			w.cursor = confirmRowDays
		case checkpointStepConfirm:
			return nil
		}
	}
	return nil
}

func (p *seedPane) Status() string { return p.statusMessage }

func (p *seedPane) Summary() string {
	if len(p.workspaces) == 0 {
		return ""
	}
	return fmt.Sprintf("%d workspaces", len(p.workspaces))
}

func (p *seedPane) DetailView(width int) string {
	row, ok := p.currentRow()
	if !ok {
		return renderDetail(width, "Workspaces", "none loaded", nil)
	}
	if row.kind == rowAPI {
		ws, _ := p.currentWorkspace()
		return renderDetail(width, orDash(row.api.Label), "api", []detailField{
			{Label: "id", Value: row.api.ID, Style: detailPlain},
			{Label: "name", Value: orDash(row.api.Label), Style: detailPlain},
			{Label: "workspace", Value: workspaceLabel(ws), Style: detailMuted},
		})
	}
	ws, ok := p.currentWorkspace()
	if !ok {
		return renderDetail(width, "Workspaces", "none loaded", nil)
	}
	fields := make([]detailField, 0, 5)
	if ws.ID == p.activeWS {
		fields = append(fields, detailField{Label: "active", Value: "yes", Style: detailOK})
	}
	fields = append(fields,
		detailField{Label: "id", Value: ws.ID, Style: detailPlain},
		detailField{Label: "slug", Value: orDash(ws.Slug), Style: detailPlain},
		detailField{Label: "name", Value: orDash(ws.Name), Style: detailPlain},
		detailField{Label: "stripe", Value: orDash(ws.StripeCustomerID), Style: detailMuted},
	)
	return renderDetail(width, workspaceLabel(ws), "workspace", fields)
}

func (p *seedPane) KeyHints() []string {
	if p.checkpoint != nil {
		return p.checkpointKeyHints()
	}
	if p.seedConfirm != nil {
		return p.seedConfirmKeyHints()
	}
	return p.keys().hints()
}

type seedLocalDoneMsg struct {
	result devseed.LocalResult
	err    error
}

type seedExecMsg struct {
	output string
	err    error
}

func (p *seedPane) execSeed(args ...string) app.Cmd {
	p.busy = true
	p.busyText = "Running seed..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		out, err := execUnkey(args...)
		return seedExecMsg{output: out, err: err}
	})
}

func (p *seedPane) View() string {
	if p.checkpoint != nil {
		return p.checkpointView()
	}
	if p.seedConfirm != nil {
		return p.seedConfirmView()
	}
	if p.loading {
		return p.spinner.View() + " " + dimStyle.Render("Loading workspaces...")
	}
	if p.loadError != "" {
		return errStyle.Render(p.loadError) + "\n" +
			dimStyle.Render("Check MySQL is up and UNKEY_DATABASE_PRIMARY is set. Press r to retry.")
	}

	var b strings.Builder
	if p.busy {
		b.WriteString(p.spinner.View() + " " + p.busyText + "\n\n")
	}
	if len(p.workspaces) == 0 {
		b.WriteString(warnStyle.Render("No workspaces in local MySQL.") + "\n" +
			dimStyle.Render("Press l to seed local (uses env vars from dev/.env)."))
		return b.String()
	}

	b.WriteString(sectionTitle("Workspaces"))
	b.WriteString("\n")
	b.WriteString(p.viewTree())
	return b.String()
}

// viewTree renders the flattened workspace/API rows with expand markers and a
// selection bar on the row under the cursor.
func (p *seedPane) viewTree() string {
	var b strings.Builder
	rows := p.visibleRows()
	rowW := barWidth(p.width, 58)
	for i, r := range rows {
		b.WriteString(p.renderSeedRow(r, i == p.cursor, rowW))
		b.WriteString("\n")
	}
	return b.String()
}

func (p *seedPane) renderSeedRow(r seedRow, selected bool, rowW int) string {
	switch r.kind {
	case rowAPILoading:
		return dimStyle.Render("   " + p.spinner.View() + " loading APIs...")
	case rowAPIEmpty:
		return dimStyle.Render("   └ no APIs — seed one with l")
	case rowWorkspace, rowAPI:
		// handled below
	}

	var marker, name, id string
	indent := ""
	if r.kind == rowWorkspace {
		marker = "▸"
		if p.expanded[r.ws.ID] {
			marker = "▾"
		}
		name = firstNonEmpty(r.ws.Name, r.ws.Slug, r.ws.ID)
		if r.ws.ID == p.activeWS {
			name += " · active"
		}
		id = r.ws.ID
	} else {
		indent = "   "
		marker = "├"
		name = firstNonEmpty(r.api.Label, r.api.ID)
		id = r.api.ID
	}

	left := indent + marker + " " + padPlain(name, 26)
	if selected {
		return selectedRowStyle.Render(padToWidth(left+"  "+id, rowW))
	}
	return left + "  " + dimStyle.Render(id)
}
