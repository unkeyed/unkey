package ui

import (
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type procsPane struct {
	rows     []procStatus
	cursor   int
	width    int
	height   int
	message  string
	logName  string // non-empty while the log view for that proc is open
	logLines []string
}

func newProcsPane() *procsPane {
	return &procsPane{} //nolint:exhaustruct
}

func (p *procsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *procsPane) InitCmd() app.Cmd { return loadProcsCmd() }

func (p *procsPane) Dispatch(msg app.Msg) (bool, app.Cmd) {
	switch msg := msg.(type) {
	case procsMsg:
		p.rows = msg.rows
		p.cursor = moveCursor(p.cursor, 0, len(p.rows))
		if p.logName != "" {
			return true, readProcLogCmd(p.logName)
		}
		return true, nil
	case procLogMsg:
		if msg.name == p.logName {
			p.logLines = msg.lines
		}
		return true, nil
	}
	return false, nil
}

func (p *procsPane) Update(msg app.Msg) app.Cmd {
	key, ok := msg.(app.KeyMsg)
	if !ok {
		return nil
	}
	if p.logName != "" {
		switch {
		case keyEsc(key), keyEnter(key):
			p.logName = ""
			p.logLines = nil
		case key.String() == "r":
			return readProcLogCmd(p.logName)
		}
		return nil
	}
	switch {
	case keyUp(key):
		p.cursor = moveCursor(p.cursor, -1, len(p.rows))
		return nil
	case keyDown(key):
		p.cursor = moveCursor(p.cursor, 1, len(p.rows))
		return nil
	}
	if cmd, ok := p.keys().handle(key); ok {
		return cmd
	}
	return nil
}

func (p *procsPane) keys() keyList {
	return keyList{
		{bind: app.Binding{Keys: []string{"enter"}, Help: "logs"}, do: func() app.Cmd {
			sel, ok := p.selected()
			if !ok {
				return nil
			}
			p.logName = sel.Name
			p.logLines = nil
			return readProcLogCmd(sel.Name)
		}},
		{bind: app.Binding{Keys: []string{"s"}, Help: "stop"}, do: func() app.Cmd {
			return p.signalSelected(syscall.SIGTERM, "Sent SIGTERM to ")
		}},
		{bind: app.Binding{Keys: []string{"k"}, Help: "kill"}, do: func() app.Cmd {
			return p.signalSelected(syscall.SIGKILL, "Killed ")
		}},
		{bind: app.Binding{Keys: []string{"c"}, Help: "clear exited"}, do: func() app.Cmd {
			cleared := clearExitedProcs()
			p.message = strconv.Itoa(cleared) + " exited cleared"
			return loadProcsCmd()
		}},
		{bind: app.Binding{Keys: []string{"r"}, Help: "refresh"}, do: func() app.Cmd {
			return loadProcsCmd()
		}},
	}
}

func (p *procsPane) signalSelected(sig syscall.Signal, verb string) app.Cmd {
	sel, ok := p.selected()
	if !ok {
		return nil
	}
	if err := signalManagedProc(sel, sig); err != nil {
		p.message = err.Error()
		return nil
	}
	p.message = verb + sel.Name
	return loadProcsCmd()
}

func (p *procsPane) selected() (procStatus, bool) {
	if p.cursor < 0 || p.cursor >= len(p.rows) {
		return procStatus{}, false //nolint:exhaustruct
	}
	return p.rows[p.cursor], true
}

func (p *procsPane) Status() string { return p.message }

func (p *procsPane) Summary() string {
	if len(p.rows) == 0 {
		return ""
	}
	running := 0
	for _, st := range p.rows {
		if st.Running {
			running++
		}
	}
	return strconv.Itoa(running) + " running · " + strconv.Itoa(len(p.rows)) + " total"
}

func (p *procsPane) DetailView(width int) string {
	st, ok := p.selected()
	if !ok {
		return renderDetail(width, "Processes", "nothing started yet", nil)
	}
	status, sstyle := "exited", detailMuted
	uptime := "—"
	if st.Running {
		status, sstyle = "running", detailOK
		uptime = formatUptime(time.Since(st.StartedAt))
	}
	fields := []detailField{
		{Label: "status", Value: status, Style: sstyle},
		{Label: "pid", Value: strconv.Itoa(st.Pid), Style: detailPlain},
		{Label: "uptime", Value: uptime, Style: detailPlain},
		{Label: "command", Value: strings.Join(st.Args, " "), Style: detailPlain},
		{Label: "log", Value: st.LogPath, Style: detailMuted},
	}
	return renderDetail(width, st.Name, status, fields)
}

func (p *procsPane) KeyHints() []string {
	if p.logName != "" {
		return []string{
			renderKeyHint("r", "refresh"),
			renderKeyHint("esc", "back"),
		}
	}
	return p.keys().hints()
}

func (p *procsPane) View() string {
	if p.logName != "" {
		return p.logView()
	}
	if len(p.rows) == 0 {
		return dimStyle.Render("No managed processes.") + "\n" +
			dimStyle.Render("Start one from the Stack or GitHub tab; it will show up here.") + "\n\n" +
			dimStyle.Render(ttyTaskNote)
	}
	var b strings.Builder
	b.WriteString(sectionTitle("Processes"))
	b.WriteString("\n")
	cols := []tableColumn{
		{Title: "Name", Width: 14},
		{Title: "Status", Width: 8},
		{Title: "PID", Width: 7},
		{Title: "Uptime", Width: 8},
		{Title: "Command", Width: 32, Dim: true},
	}
	rows := make([][]string, 0, len(p.rows))
	for _, st := range p.rows {
		status := "exited"
		uptime := "-"
		if st.Running {
			status = "running"
			uptime = formatUptime(time.Since(st.StartedAt))
		}
		rows = append(rows, []string{
			st.Name,
			status,
			strconv.Itoa(st.Pid),
			uptime,
			strings.Join(st.Args, " "),
		})
	}
	b.WriteString(renderTable(p.width, cols, rows, p.cursor))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(ttyTaskNote))
	return b.String()
}

// ttyTaskNote explains why sudo tasks (tunnel) never appear in this list.
const ttyTaskNote = "Sudo tasks (tunnel) open in their own terminal window, not here."

func (p *procsPane) logView() string {
	var b strings.Builder
	b.WriteString(sectionTitle("Logs: " + p.logName))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(procLogPath(p.logName)))
	b.WriteString("\n\n")
	lines := p.logLines
	// Fit inside the frame; overflow at the top is old output, overflow at
	// the bottom would be clipped by MaxHeight and hide the newest lines.
	visible := p.height - 6
	if visible < 4 {
		visible = 4
	}
	if len(lines) > visible {
		lines = lines[len(lines)-visible:]
	}
	maxW := effectiveRowWidth(p.width)
	for _, line := range lines {
		b.WriteString(truncate(line, maxW))
		b.WriteString("\n")
	}
	if len(lines) == 0 {
		b.WriteString(dimStyle.Render("no log output yet"))
	}
	return b.String()
}

func formatUptime(d time.Duration) string {
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + "s"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m"
}
