package ui

import "github.com/unkeyed/unkey/pkg/tui/app"

type stackConfirm struct {
	task   stackTask
	cursor int
}

func (p *stackPane) stackConfirmLines() []confirmLine {
	if p.confirm == nil {
		return nil
	}
	t := p.confirm.task
	effect := "Starts in background"
	switch t.task {
	case "down":
		effect = "Stops Tilt and deletes minikube"
	case "nuke":
		effect = "Deletes Docker data and build caches"
	}
	return []confirmLine{
		{Label: "task", Value: t.name, Scope: true},
		{Label: "effect", Value: effect},
		{Label: "command", Value: "mise run " + t.task},
	}
}

func (p *stackPane) updateStackConfirm(key app.KeyMsg) app.Cmd {
	if p.confirm == nil {
		return nil
	}
	lines := p.stackConfirmLines()
	nav := navigateConfirm(key, p.confirm.cursor, lines)
	if nav.Esc {
		p.confirm = nil
		return nil
	}
	p.confirm.cursor = nav.Cursor
	changeScope, run := handleConfirmEnter(nav, lines)
	if changeScope {
		p.confirm = nil
		return nil
	}
	if run {
		task := p.confirm.task
		p.confirm = nil
		return p.runTask(task)
	}
	return nil
}

func (p *stackPane) confirmView() string {
	if p.confirm == nil {
		return ""
	}
	return renderConfirmScreen(p.width, "Run stack task", "", p.stackConfirmLines(), p.confirm.cursor)
}

func (p *stackPane) confirmKeyHints() []string {
	return confirmKeyHints(p.stackConfirmLines(), p.confirm.cursor)
}
