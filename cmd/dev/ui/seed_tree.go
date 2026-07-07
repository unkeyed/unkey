package ui

import (
	"github.com/unkeyed/unkey/pkg/tui/app"
)

// The seed list is a flat cursor over a tree: workspace rows that expand in
// place to show their APIs as indented children, so picking an API for
// verifications never leaves the list or asks you to type an id.

type seedRowKind int

const (
	rowWorkspace seedRowKind = iota
	rowAPI
	rowAPILoading
	rowAPIEmpty
)

type seedRow struct {
	kind seedRowKind
	ws   workspaceRow
	api  namedChoice
	wsID string
}

// visibleRows flattens workspaces and the APIs of any expanded workspace into
// the single list the cursor moves over.
func (p *seedPane) visibleRows() []seedRow {
	var rows []seedRow
	for _, ws := range p.workspaces {
		rows = append(rows, seedRow{kind: rowWorkspace, ws: ws, wsID: ws.ID}) //nolint:exhaustruct
		if !p.expanded[ws.ID] {
			continue
		}
		if p.apiLoading[ws.ID] {
			rows = append(rows, seedRow{kind: rowAPILoading, wsID: ws.ID}) //nolint:exhaustruct
			continue
		}
		apis := p.apis[ws.ID]
		if len(apis) == 0 {
			rows = append(rows, seedRow{kind: rowAPIEmpty, wsID: ws.ID}) //nolint:exhaustruct
			continue
		}
		for _, a := range apis {
			rows = append(rows, seedRow{kind: rowAPI, api: a, wsID: ws.ID}) //nolint:exhaustruct
		}
	}
	return rows
}

func (p *seedPane) currentRow() (seedRow, bool) {
	rows := p.visibleRows()
	if p.cursor < 0 || p.cursor >= len(rows) {
		return seedRow{}, false //nolint:exhaustruct
	}
	return rows[p.cursor], true
}

// currentWorkspace is the workspace the current row belongs to: itself for a
// workspace row, the parent for an API/loading/empty child row.
func (p *seedPane) currentWorkspace() (workspaceRow, bool) {
	row, ok := p.currentRow()
	if !ok {
		return workspaceRow{}, false //nolint:exhaustruct
	}
	if row.kind == rowWorkspace {
		return row.ws, true
	}
	for _, ws := range p.workspaces {
		if ws.ID == row.wsID {
			return ws, true
		}
	}
	return workspaceRow{}, false //nolint:exhaustruct
}

func (p *seedPane) clampCursor() {
	p.cursor = moveCursor(p.cursor, 0, len(p.visibleRows()))
}

func (p *seedPane) apiLoadingAny() bool {
	for _, loading := range p.apiLoading {
		if loading {
			return true
		}
	}
	return false
}

// toggleExpand expands or collapses the workspace under the cursor, loading its
// APIs on first expand.
func (p *seedPane) toggleExpand() app.Cmd {
	row, ok := p.currentRow()
	if !ok || row.kind != rowWorkspace {
		return nil
	}
	id := row.ws.ID
	if p.expanded[id] {
		p.expanded[id] = false
		p.clampCursor()
		return nil
	}
	p.expanded[id] = true
	if p.apis[id] == nil && !p.apiLoading[id] {
		p.apiLoading[id] = true
		return withSpinnerTick(p.spinner, listAPIsCmd(row.ws))
	}
	return nil
}

// collapseLeft collapses an expanded workspace, or from an API child jumps back
// to its parent workspace and collapses it.
func (p *seedPane) collapseLeft() {
	row, ok := p.currentRow()
	if !ok {
		return
	}
	if row.kind == rowWorkspace {
		if p.expanded[row.ws.ID] {
			p.expanded[row.ws.ID] = false
		}
		return
	}
	p.expanded[row.wsID] = false
	for i, r := range p.visibleRows() {
		if r.kind == rowWorkspace && r.ws.ID == row.wsID {
			p.cursor = i
			return
		}
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
