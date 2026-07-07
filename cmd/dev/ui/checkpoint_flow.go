package ui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

type checkpointStep int

const (
	checkpointStepProject checkpointStep = iota
	checkpointStepApp
	checkpointStepEnv
	checkpointStepConfirm
)

const (
	confirmRowProject  = 0
	confirmRowApp      = 1
	confirmRowEnv      = 2
	confirmRowDays     = 3
	confirmRowHours    = 4
	confirmRowTick     = 5
	confirmRowReplicas = 6
	confirmRowVCPU     = 7
	confirmRowMemory   = 8
	confirmRowDisk     = 9
	confirmRowEgress   = 10
)

var checkpointTickOptions = []time.Duration{
	30 * time.Second,
	60 * time.Second,
	90 * time.Second,
}

var checkpointVCPUOptions = []string{"0.25", "0.5", "1", "2", "4"}

var checkpointMemoryOptions = []string{"256Mi", "512Mi", "1Gi", "2Gi", "4Gi"}

// Empty string means "none": the flag is omitted and the seeder skips that meter.
var checkpointDiskOptions = []string{"", "1Gi", "5Gi", "10Gi", "20Gi"}

var checkpointEgressOptions = []string{"", "100Mi", "512Mi", "1Gi", "2Gi", "10Gi"}

func optionOrNone(opts []string, idx int) string {
	if idx < 0 || idx >= len(opts) {
		return ""
	}
	return opts[idx]
}

func displayOption(opts []string, idx int) string {
	v := optionOrNone(opts, idx)
	if v == "" {
		return "none"
	}
	return v
}

type namedChoice struct {
	ID    string
	Label string
}

type checkpointWizard struct {
	workspaceID string
	step        checkpointStep
	choices     []namedChoice
	cursor      int
	loading     bool
	projectID   string
	projectName string
	appID       string
	appName     string
	envID       string
	envName     string
	days        int
	hoursPerDay int
	tickIdx     int
	replicas    int
	vcpuIdx     int
	memIdx      int
	diskIdx     int
	egressIdx   int
}

type checkpointChoicesMsg struct {
	step    checkpointStep
	choices []namedChoice
	err     error
}

func defaultCheckpointWizard(workspaceID string) *checkpointWizard {
	return &checkpointWizard{ //nolint:exhaustruct
		workspaceID: workspaceID,
		step:        checkpointStepProject,
		loading:     true,
		days:        30,
		hoursPerDay: 24,
		tickIdx:     1,
		replicas:    1,
		vcpuIdx:     1,
		memIdx:      1,
		diskIdx:     0,
		egressIdx:   0,
	}
}

func (w *checkpointWizard) tick() time.Duration {
	if w.tickIdx < 0 || w.tickIdx >= len(checkpointTickOptions) {
		return 60 * time.Second
	}
	return checkpointTickOptions[w.tickIdx]
}

func (p *seedPane) startCheckpointWizard() app.Cmd {
	ws, ok := p.currentWorkspace()
	if !ok {
		p.statusMessage = "select a workspace first"
		return nil
	}
	p.checkpoint = defaultCheckpointWizard(ws.ID)
	return p.loadCheckpointChoices(checkpointStepProject)
}

func (p *seedPane) loadCheckpointChoices(step checkpointStep) app.Cmd {
	return withSpinnerTick(p.spinner, func() app.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		var (
			rows []namedChoice
			err  error
		)
		switch step {
		case checkpointStepProject:
			rows, err = listProjects(ctx, p.checkpoint.workspaceID)
		case checkpointStepApp:
			rows, err = listApps(ctx, p.checkpoint.projectID)
		case checkpointStepEnv:
			rows, err = listEnvironments(ctx, p.checkpoint.appID)
		case checkpointStepConfirm:
			return checkpointChoicesMsg{step: step, choices: nil, err: fmt.Errorf("no choices for confirm step")}
		}
		return checkpointChoicesMsg{step: step, choices: rows, err: err}
	})
}

func listProjects(ctx context.Context, workspaceID string) ([]namedChoice, error) {
	return queryNamedChoices(ctx,
		"SELECT id, slug FROM projects WHERE workspace_id = ? ORDER BY created_at DESC",
		workspaceID,
	)
}

func listApps(ctx context.Context, projectID string) ([]namedChoice, error) {
	return queryNamedChoices(ctx,
		"SELECT id, slug FROM apps WHERE project_id = ? ORDER BY created_at DESC",
		projectID,
	)
}

func listEnvironments(ctx context.Context, appID string) ([]namedChoice, error) {
	return queryNamedChoices(ctx,
		"SELECT id, slug FROM environments WHERE app_id = ? ORDER BY created_at DESC",
		appID,
	)
}

func queryNamedChoices(ctx context.Context, query string, args ...any) ([]namedChoice, error) {
	database, err := db.New(db.Config{
		PrimaryDSN:  databaseDSN(),
		ReadOnlyDSN: databaseDSN(),
		Tags:        sqlcomment.Disabled(),
	})
	if err != nil {
		return nil, err
	}
	dbRows, err := database.RO().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = dbRows.Close() }()
	var out []namedChoice
	for dbRows.Next() {
		var row namedChoice
		if err := dbRows.Scan(&row.ID, &row.Label); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, dbRows.Err()
}

func (p *seedPane) runCheckpoints() app.Cmd {
	w := p.checkpoint
	if w == nil {
		return nil
	}
	p.busy = true
	p.busyText = "Seeding checkpoints..."
	p.checkpoint = nil
	args := []string{
		"dev", "seed", "checkpoints",
		"--workspace=" + w.workspaceID,
		"--project=" + w.projectID,
		"--app=" + w.appID,
		"--environment=" + w.envID,
		"--days=" + strconv.Itoa(w.days),
		"--hours-per-day=" + strconv.Itoa(w.hoursPerDay),
		"--tick=" + w.tick().String(),
		"--replicas=" + strconv.Itoa(w.replicas),
		"--vcpu=" + optionOrNone(checkpointVCPUOptions, w.vcpuIdx),
		"--memory=" + optionOrNone(checkpointMemoryOptions, w.memIdx),
	}
	if disk := optionOrNone(checkpointDiskOptions, w.diskIdx); disk != "" {
		args = append(args, "--disk="+disk)
	}
	if egress := optionOrNone(checkpointEgressOptions, w.egressIdx); egress != "" {
		args = append(args, "--egress-per-day="+egress)
	}
	return withSpinnerTick(p.spinner, func() app.Msg {
		out, err := execUnkey(args...)
		return seedExecMsg{output: out, err: err}
	})
}

func (p *seedPane) checkpointBack() {
	if p.checkpoint == nil {
		return
	}
	switch p.checkpoint.step {
	case checkpointStepConfirm:
		p.checkpoint.step = checkpointStepEnv
		p.checkpoint.cursor = 0
	case checkpointStepEnv:
		p.checkpoint.step = checkpointStepApp
		p.checkpoint.appID = ""
		p.checkpoint.appName = ""
		p.checkpoint.envID = ""
		p.checkpoint.envName = ""
		p.checkpoint.cursor = 0
		p.checkpoint.loading = true
	case checkpointStepApp:
		p.checkpoint.step = checkpointStepProject
		p.checkpoint.projectID = ""
		p.checkpoint.projectName = ""
		p.checkpoint.appID = ""
		p.checkpoint.appName = ""
		p.checkpoint.envID = ""
		p.checkpoint.envName = ""
		p.checkpoint.cursor = 0
		p.checkpoint.loading = true
	case checkpointStepProject:
		p.checkpoint = nil
	}
}

func (p *seedPane) reopenCheckpointScope(row int) app.Cmd {
	w := p.checkpoint
	if w == nil {
		return nil
	}
	switch row {
	case confirmRowProject:
		w.step = checkpointStepProject
		w.projectID = ""
		w.projectName = ""
		w.appID = ""
		w.appName = ""
		w.envID = ""
		w.envName = ""
		w.cursor = 0
		w.loading = true
		return p.loadCheckpointChoices(checkpointStepProject)
	case confirmRowApp:
		w.step = checkpointStepApp
		w.appID = ""
		w.appName = ""
		w.envID = ""
		w.envName = ""
		w.cursor = 0
		w.loading = true
		return p.loadCheckpointChoices(checkpointStepApp)
	case confirmRowEnv:
		w.step = checkpointStepEnv
		w.envID = ""
		w.envName = ""
		w.cursor = 0
		w.loading = true
		return p.loadCheckpointChoices(checkpointStepEnv)
	default:
		return nil
	}
}

func (p *seedPane) adjustCheckpointConfirm(delta int) {
	w := p.checkpoint
	if w == nil {
		return
	}
	lines := w.confirmLines()
	if w.cursor >= len(lines) || !lines[w.cursor].Adjustable {
		return
	}
	switch w.cursor {
	case confirmRowDays:
		w.days = clampInt(w.days+delta, 1, 90)
	case confirmRowHours:
		w.hoursPerDay = clampInt(w.hoursPerDay+delta, 1, 24)
	case confirmRowTick:
		w.tickIdx = clampInt(w.tickIdx+delta, 0, len(checkpointTickOptions)-1)
	case confirmRowReplicas:
		w.replicas = clampInt(w.replicas+delta, 1, 8)
	case confirmRowVCPU:
		w.vcpuIdx = clampInt(w.vcpuIdx+delta, 0, len(checkpointVCPUOptions)-1)
	case confirmRowMemory:
		w.memIdx = clampInt(w.memIdx+delta, 0, len(checkpointMemoryOptions)-1)
	case confirmRowDisk:
		w.diskIdx = clampInt(w.diskIdx+delta, 0, len(checkpointDiskOptions)-1)
	case confirmRowEgress:
		w.egressIdx = clampInt(w.egressIdx+delta, 0, len(checkpointEgressOptions)-1)
	}
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (w *checkpointWizard) confirmLines() []confirmLine {
	return []confirmLine{
		{Label: "project", Value: w.projectName, Scope: true},
		{Label: "app", Value: w.appName, Scope: true},
		{Label: "environment", Value: w.envName, Scope: true},
		{Label: "days", Value: strconv.Itoa(w.days), Adjustable: true},
		{Label: "hours/day", Value: strconv.Itoa(w.hoursPerDay), Adjustable: true},
		{Label: "tick", Value: w.tick().String(), Adjustable: true},
		{Label: "replicas", Value: strconv.Itoa(w.replicas), Adjustable: true},
		{Label: "vcpu", Value: displayOption(checkpointVCPUOptions, w.vcpuIdx), Adjustable: true},
		{Label: "memory", Value: displayOption(checkpointMemoryOptions, w.memIdx), Adjustable: true},
		{Label: "disk", Value: displayOption(checkpointDiskOptions, w.diskIdx), Adjustable: true},
		{Label: "egress/day", Value: displayOption(checkpointEgressOptions, w.egressIdx), Adjustable: true},
	}
}

func (p *seedPane) updateCheckpointConfirm(key app.KeyMsg) app.Cmd {
	w := p.checkpoint
	if w == nil || w.loading {
		return nil
	}
	lines := w.confirmLines()
	nav := navigateConfirm(key, w.cursor, lines)
	if nav.Esc {
		p.checkpointBack()
		if p.checkpoint != nil && p.checkpoint.loading {
			return p.loadCheckpointChoices(p.checkpoint.step)
		}
		return nil
	}
	w.cursor = nav.Cursor
	if nav.Adjust != 0 {
		p.adjustCheckpointConfirm(nav.Adjust)
		return nil
	}
	changeScope, run := handleConfirmEnter(nav, lines)
	if changeScope {
		return p.reopenCheckpointScope(w.cursor)
	}
	if run {
		return p.runCheckpoints()
	}
	return nil
}

func (p *seedPane) checkpointView() string {
	w := p.checkpoint
	if w == nil {
		return ""
	}
	if w.loading {
		return p.spinner.View() + " " + dimStyle.Render("Loading...")
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Seed checkpoints"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("workspace " + truncate(w.workspaceID, 28)))
	b.WriteString("\n\n")

	if w.step == checkpointStepConfirm {
		lines := w.confirmLines()
		return renderConfirmScreen(p.width, "Seed checkpoints", "workspace "+truncate(w.workspaceID, 28), lines, w.cursor)
	}

	title := "Select"
	switch w.step {
	case checkpointStepProject:
		title = "Select project"
	case checkpointStepApp:
		title = "Select app"
	case checkpointStepEnv:
		title = "Select environment"
	case checkpointStepConfirm:
		title = "Confirm"
	}

	if len(w.choices) == 0 {
		b.WriteString(warnStyle.Render("Nothing here yet. Seed a project first (l local)."))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Esc back"))
		return b.String()
	}

	cols := []tableColumn{ //nolint:exhaustruct
		{Title: title, Width: 20},
		{Title: "ID", Width: 26, Dim: true},
	}
	rows := make([][]string, 0, len(w.choices))
	for _, c := range w.choices {
		rows = append(rows, []string{c.Label, c.ID})
	}
	b.WriteString(renderTable(p.width, cols, rows, w.cursor))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Enter select   Esc back"))
	return b.String()
}

func (p *seedPane) checkpointKeyHints() []string {
	w := p.checkpoint
	if w != nil && w.step == checkpointStepConfirm {
		return confirmKeyHints(w.confirmLines(), w.cursor)
	}
	return []string{
		renderKeyHint("↑/↓", "nav"),
		renderKeyHint("enter", "select"),
		renderKeyHint("esc", "back"),
	}
}
