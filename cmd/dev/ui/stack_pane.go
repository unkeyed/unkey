package ui

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type stackTask struct {
	task string
	name string
	desc string
	// confirm gates destructive tasks behind a confirm screen; everything
	// else starts straight from Enter.
	confirm bool
	// stdin is canned prompt input for tasks that read from the terminal
	// (they run detached, so nobody can answer interactively).
	stdin string
	// needsTTY tasks require an interactive terminal (e.g. a sudo password
	// prompt) and are launched in a new terminal window instead of captured
	// as a headless managed process, which has no tty for the prompt.
	needsTTY bool
}

type stackPane struct {
	tasks    []stackTask
	cursor   int
	width    int
	height   int
	message  string
	confirm  *stackConfirm
	busy     bool
	busyText string
	spinner  app.Spinner
	// ports is the last async check result; nil until the first one lands.
	// View must never dial itself: it runs on every keypress.
	ports map[string]bool
}

func newStackPane() *stackPane {
	s := app.NewSpinner()
	return &stackPane{ //nolint:exhaustruct
		tasks: []stackTask{
			{task: "dev", name: "Full stack", desc: "mise run dev", confirm: false, stdin: "", needsTTY: false},
			{task: "dashboard", name: "Dashboard only", desc: "mise run dashboard", confirm: false, stdin: "", needsTTY: false},
			// tunnel binds privileged ports 80/443, so it needs sudo (a tty).
			{task: "tunnel", name: "HTTPS tunnel", desc: "mise run tunnel · sudo", confirm: false, stdin: "", needsTTY: true},
			{task: "generate", name: "Generate code", desc: "mise run generate", confirm: false, stdin: "", needsTTY: false},
			{task: "down", name: "Tear down", desc: "mise run down", confirm: true, stdin: "", needsTTY: false},
			// nuke prompts per category with yes as the default; blank lines
			// accept everything, which is what the confirm screen states.
			{task: "nuke", name: "Nuke caches", desc: "mise run nuke", confirm: true, stdin: strings.Repeat("\n", 16), needsTTY: false},
		},
		spinner: s,
	}
}

func (p *stackPane) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *stackPane) InitCmd() app.Cmd { return checkPortsCmd() }

func (p *stackPane) Dispatch(msg app.Msg) (bool, app.Cmd) {
	switch msg := msg.(type) {
	case portsMsg:
		p.ports = msg.open
		return true, nil
	case stackMsg:
		p.busy = false
		if msg.err != nil {
			p.message = msg.err.Error()
			if msg.message != "" {
				p.message += "\n" + msg.message
			}
			return true, nil
		}
		p.message = msg.message
		// Refresh the Procs tab so the new process shows up right away.
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

func (p *stackPane) Update(msg app.Msg) app.Cmd {
	if p.confirm != nil {
		if key, ok := msg.(app.KeyMsg); ok {
			return p.updateStackConfirm(key)
		}
		return nil
	}
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

func (p *stackPane) keys() keyList {
	return keyList{
		{bind: app.Binding{Keys: []string{"enter"}, Help: "run"}, do: func() app.Cmd {
			task := p.tasks[p.cursor]
			if task.confirm {
				p.confirm = &stackConfirm{task: task, cursor: 0}
				return nil
			}
			return p.runTask(task)
		}},
	}
}

func (p *stackPane) Status() string { return p.message }

func (p *stackPane) Summary() string { return "" }

// DetailView is empty: the stack tab is a fixed menu with no per-row record.
func (p *stackPane) DetailView(int) string { return "" }

func (p *stackPane) KeyHints() []string {
	if p.confirm != nil {
		return p.confirmKeyHints()
	}
	return p.keys().hints()
}

func (p *stackPane) runTask(task stackTask) app.Cmd {
	p.busy = true
	p.busyText = "Starting " + task.name + "..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		// Tasks that need a tty (sudo) run in a new terminal window; a headless
		// managed process has no terminal for the password prompt.
		if task.needsTTY {
			if err := runInTerminal(miseArgv(task.task)); err != nil {
				return stackMsg{message: "", err: err}
			}
			return stackMsg{message: task.name + " opened in a new terminal · enter your sudo password there", err: nil}
		}
		if err := startManagedProc(task.task, miseArgv(task.task), task.stdin); err != nil {
			return stackMsg{message: "", err: err}
		}
		return stackMsg{message: "Started " + task.name + " · logs in Procs (5)", err: nil}
	})
}

type stackMsg struct {
	message string
	err     error
}

func (p *stackPane) View() string {
	if p.confirm != nil {
		return p.confirmView()
	}
	var b strings.Builder
	if p.busy {
		b.WriteString(p.spinner.View() + " " + p.busyText + "\n\n")
	}
	items := make([]menuItem, len(p.tasks))
	for i, t := range p.tasks {
		items[i] = menuItem{Name: t.name, Desc: t.desc}
	}
	b.WriteString(renderMenuList(p.width, "Stack tasks", items, p.cursor))
	b.WriteString("\n")
	b.WriteString(sectionTitle("Ports"))
	b.WriteString("\n")
	for _, port := range stackPorts() {
		mark := "checking"
		style := dimStyle
		if p.ports != nil {
			if p.ports[port.Port] {
				mark = "up"
				style = okStyle
			} else {
				mark = "down"
			}
		}
		line := fmt.Sprintf("  %-12s :%-5s %s", port.Name, port.Port, mark)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}
