package ui

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type githubTask struct {
	args string
	name string
	desc string
	// proc is the managed-process name for background tasks; empty means the
	// task runs to completion in the foreground.
	proc string
}

type githubPane struct {
	tasks    []githubTask
	cursor   int
	width    int
	height   int
	message  string
	busy     bool
	busyText string
	spinner  app.Spinner
}

func newGithubPane() *githubPane {
	s := app.NewSpinner()
	return &githubPane{ //nolint:exhaustruct
		tasks: []githubTask{
			{args: "dev github setup", name: "Setup GitHub app", desc: "Browser flow on :9999", proc: "github-setup"},
			{args: "dev github tunnel", name: "Webhook tunnel", desc: "ngrok tunnel for webhooks", proc: "github-tunnel"},
			{args: "dev github trigger-webhook", name: "Trigger webhook", desc: "Send test webhook event", proc: ""},
		},
		spinner: s,
	}
}

func (p *githubPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *githubPane) InitCmd() app.Cmd { return nil }

func (p *githubPane) Dispatch(msg app.Msg) (bool, app.Cmd) {
	switch msg := msg.(type) {
	case githubMsg:
		p.busy = false
		p.message = msg.message
		return true, loadProcsCmd()
	case app.SpinnerTickMsg:
		if p.busy {
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

func (p *githubPane) Update(msg app.Msg) app.Cmd {
	key, ok := msg.(app.KeyMsg)
	if !ok || p.busy {
		return nil
	}
	switch {
	case keyUp(key):
		p.cursor = moveCursor(p.cursor, -1, len(p.tasks))
		return nil
	case keyDown(key):
		p.cursor = moveCursor(p.cursor, 1, len(p.tasks))
		return nil
	}
	if cmd, ok := p.keys().handle(key); ok {
		return cmd
	}
	return nil
}

func (p *githubPane) keys() keyList {
	return keyList{
		{bind: app.Binding{Keys: []string{"enter"}, Help: "run"}, do: func() app.Cmd {
			return p.runTask(p.tasks[p.cursor])
		}},
	}
}

func (p *githubPane) runTask(task githubTask) app.Cmd {
	p.busy = true
	p.busyText = "Running " + task.name + "..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		parts := strings.Fields(task.args)
		if task.proc != "" {
			if err := startManagedProc(task.proc, unkeyArgv(parts...), ""); err != nil {
				return githubMsg{message: task.name + " failed: " + err.Error()}
			}
			return githubMsg{message: "Started " + task.name + " · logs in Procs (5)"}
		}
		out, err := execUnkey(parts...)
		if err != nil {
			return githubMsg{message: task.name + " failed: " + err.Error() + "\n" + out}
		}
		if out != "" {
			return githubMsg{message: out}
		}
		return githubMsg{message: "Done"}
	})
}

func (p *githubPane) Status() string { return p.message }

func (p *githubPane) Summary() string { return "" }

// DetailView is empty: the github tab is a fixed menu with no per-row record.
func (p *githubPane) DetailView(int) string { return "" }

func (p *githubPane) KeyHints() []string {
	return p.keys().hints()
}

type githubMsg struct {
	message string
}

func (p *githubPane) View() string {
	prefix := ""
	if p.busy {
		prefix = p.spinner.View() + " " + p.busyText + "\n\n"
	}
	items := make([]menuItem, len(p.tasks))
	for i, t := range p.tasks {
		items[i] = menuItem{Name: t.name, Desc: t.desc}
	}
	return prefix + renderMenuList(p.width, "GitHub dev", items, p.cursor)
}
