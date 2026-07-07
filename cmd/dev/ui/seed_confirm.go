package ui

import (
	"context"
	"strconv"

	"github.com/unkeyed/unkey/pkg/tui/app"
)

type seedActionKind int

const (
	seedActionLocal seedActionKind = iota
	seedActionFrontline
	seedActionVerifications
)

type seedActionConfirm struct {
	action    seedActionKind
	workspace workspaceRow
	hasWS     bool
	apiID     string
	apiName   string
	countIdx  int
	cursor    int
}

// verifCountOptions is the num-verifications stepper. num-verifications is a
// required flag with no default, so it must always be set (the old flow never
// passed it and silently failed).
var verifCountOptions = []int{100, 1000, 10000, 100000, 1000000}

const defaultVerifCountIdx = 1 // 1000

// openSeedConfirm opens the confirm screen for actions that take no picked
// target (local, frontline). Verifications is driven from the inline API row.
func (p *seedPane) openSeedConfirm(action seedActionKind) app.Cmd {
	c := &seedActionConfirm{action: action, cursor: 0} //nolint:exhaustruct
	if ws, ok := p.currentWorkspace(); ok {
		c.workspace = ws
		c.hasWS = true
	}
	c.cursor = confirmFirstAdjustable(p.seedConfirmLines(c))
	p.seedConfirm = c
	return nil
}

// startVerifications opens the count confirm for an API picked inline from the
// workspace tree. No typed or cycled id, and no workspace re-ask.
func (p *seedPane) startVerifications(wsID string, api namedChoice) app.Cmd {
	ws := workspaceRow{ID: wsID} //nolint:exhaustruct
	for _, w := range p.workspaces {
		if w.ID == wsID {
			ws = w
			break
		}
	}
	c := &seedActionConfirm{ //nolint:exhaustruct
		action:    seedActionVerifications,
		workspace: ws,
		hasWS:     true,
		apiID:     api.ID,
		apiName:   api.Label,
		countIdx:  defaultVerifCountIdx,
	}
	c.cursor = confirmFirstAdjustable(p.seedConfirmLines(c))
	p.seedConfirm = c
	return nil
}

func workspaceLabel(ws workspaceRow) string {
	if ws.Slug != "" {
		return ws.Slug
	}
	if ws.Name != "" {
		return ws.Name
	}
	return ws.ID
}

func (p *seedPane) seedConfirmLines(c *seedActionConfirm) []confirmLine {
	if c == nil {
		return nil
	}
	switch c.action {
	case seedActionLocal:
		slug := "local"
		if c.hasWS && c.workspace.Slug != "" {
			slug = c.workspace.Slug
		}
		return []confirmLine{
			{Label: "action", Value: "local seed"},
			{Label: "slug", Value: slug},
			{Label: "output", Value: envOr("UNKEY_SEED_OUTPUT", "dev/.env.seed")},
		}
	case seedActionFrontline:
		return []confirmLine{{Label: "action", Value: "frontline seed"}}
	case seedActionVerifications:
		return []confirmLine{
			{Label: "api", Value: truncate(c.apiName, 18) + " · " + truncate(c.apiID, 22), Scope: true},
			{Label: "count", Value: formatCount(verifCountOptions[c.countIdx]), Adjustable: true},
		}
	}
	return nil
}

func formatCount(n int) string {
	switch {
	case n >= 1_000_000:
		return strconv.Itoa(n/1_000_000) + "M"
	case n >= 1_000:
		return strconv.Itoa(n/1_000) + "k"
	default:
		return strconv.Itoa(n)
	}
}

func (p *seedPane) updateSeedConfirm(key app.KeyMsg) app.Cmd {
	if p.seedConfirm == nil {
		return nil
	}
	c := p.seedConfirm
	lines := p.seedConfirmLines(c)
	nav := navigateConfirm(key, c.cursor, lines)
	if nav.Esc {
		// Back to the tree; the API stays highlighted so you can re-pick.
		p.seedConfirm = nil
		return nil
	}
	c.cursor = nav.Cursor
	if nav.Adjust != 0 && c.action == seedActionVerifications {
		c.countIdx = clampInt(c.countIdx+nav.Adjust, 0, len(verifCountOptions)-1)
		return nil
	}
	changeScope, run := handleConfirmEnter(nav, lines)
	if changeScope {
		// The api Scope row just returns to the tree to pick a different API.
		p.seedConfirm = nil
		return nil
	}
	if !run {
		return nil
	}
	p.seedConfirm = nil
	switch c.action {
	case seedActionLocal:
		slug := ""
		if c.hasWS && c.workspace.Slug != "" {
			slug = c.workspace.Slug
		}
		return p.seedLocalConfirmed(slug)
	case seedActionFrontline:
		return p.execSeed("dev", "seed", "frontline")
	case seedActionVerifications:
		return p.execSeed("dev", "seed", "verifications",
			"--api-id="+c.apiID,
			"--num-verifications="+strconv.Itoa(verifCountOptions[c.countIdx]),
		)
	default:
		return nil
	}
}

func (p *seedPane) seedLocalConfirmed(slug string) app.Cmd {
	p.busy = true
	p.busyText = "Seeding local..."
	return withSpinnerTick(p.spinner, func() app.Msg {
		result, err := runLocalSeed(context.Background(), slug)
		return seedLocalDoneMsg{result: result, err: err}
	})
}

func (p *seedPane) seedConfirmView() string {
	if p.seedConfirm == nil {
		return ""
	}
	return renderConfirmScreen(p.width, "Run seed", "", p.seedConfirmLines(p.seedConfirm), p.seedConfirm.cursor)
}

func (p *seedPane) seedConfirmKeyHints() []string {
	if p.seedConfirm == nil {
		return nil
	}
	return confirmKeyHints(p.seedConfirmLines(p.seedConfirm), p.seedConfirm.cursor)
}
