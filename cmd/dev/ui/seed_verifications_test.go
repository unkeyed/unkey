package ui

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/tui/app"
)

func newSeedPaneWithWorkspaces(rows []workspaceRow) *seedPane {
	p := newSeedPane()
	p.loading = false
	p.workspaces = rows
	p.SetSize(100, 28)
	return p
}

func keyString(s string) app.KeyMsg {
	return app.KeyMsg{Type: app.KeyRunes, Runes: []rune(s), Alt: false, Paste: false}
}

// Expanding a workspace loads its APIs and shows them as indented child rows.
func TestExpandWorkspaceLoadsAPIsInline(t *testing.T) {
	p := newSeedPaneWithWorkspaces([]workspaceRow{
		{ID: "ws_1", Slug: "acme", Name: "Acme", StripeCustomerID: ""},
	})
	// Right arrow expands the workspace under the cursor and kicks off a load.
	p.Update(app.KeyMsg{Type: app.KeyRight}) //nolint:exhaustruct
	require.True(t, p.expanded["ws_1"])
	require.True(t, p.apiLoading["ws_1"])

	p.Dispatch(apiChoicesMsg{workspaceID: "ws_1", choices: []namedChoice{
		{ID: "api_abc", Label: "primary"},
		{ID: "api_xyz", Label: "secondary"},
	}, err: nil})
	require.False(t, p.apiLoading["ws_1"])

	rows := p.visibleRows()
	require.Len(t, rows, 3) // workspace + 2 apis
	require.Equal(t, rowWorkspace, rows[0].kind)
	require.Equal(t, rowAPI, rows[1].kind)
	require.Equal(t, "api_abc", rows[1].api.ID)
	require.Equal(t, rowAPI, rows[2].kind)
}

// v on an API child row goes straight to the count confirm with that API.
func TestVerificationsFromAPIRow(t *testing.T) {
	p := newSeedPaneWithWorkspaces([]workspaceRow{{ID: "ws_1", Slug: "acme", Name: "Acme", StripeCustomerID: ""}})
	p.Update(app.KeyMsg{Type: app.KeyRight}) //nolint:exhaustruct
	p.Dispatch(apiChoicesMsg{workspaceID: "ws_1", choices: []namedChoice{
		{ID: "api_abc", Label: "primary"},
		{ID: "api_xyz", Label: "secondary"},
	}, err: nil})

	// Move down to the second API row and press v.
	p.Update(app.KeyMsg{Type: app.KeyDown}) //nolint:exhaustruct // -> first api
	p.Update(app.KeyMsg{Type: app.KeyDown}) //nolint:exhaustruct // -> second api
	require.Nil(t, p.seedConfirm)
	p.Update(keyString("v"))

	require.NotNil(t, p.seedConfirm, "v on an API opens the count confirm")
	require.Equal(t, "api_xyz", p.seedConfirm.apiID)
	require.Equal(t, "secondary", p.seedConfirm.apiName)

	view := p.seedConfirmView()
	require.Contains(t, view, "count")
	require.NotContains(t, view, "workspace")

	lines := p.seedConfirmLines(p.seedConfirm)
	p.seedConfirm.cursor = confirmFirstAdjustable(lines)
	p.Update(app.KeyMsg{Type: app.KeyRight}) //nolint:exhaustruct // count 1000 -> 10000
	require.Equal(t, 10000, verifCountOptions[p.seedConfirm.countIdx])

	require.Equal(t, []string{
		"dev", "seed", "verifications",
		"--api-id=api_xyz",
		"--num-verifications=10000",
	}, verificationsArgsFor(p.seedConfirm))
}

// v on a workspace row expands it and points the user at its APIs.
func TestVerificationsOnWorkspaceExpands(t *testing.T) {
	p := newSeedPaneWithWorkspaces([]workspaceRow{{ID: "ws_1", Slug: "acme", Name: "Acme", StripeCustomerID: ""}})
	p.Update(keyString("v"))
	require.True(t, p.expanded["ws_1"])
	require.Nil(t, p.seedConfirm)
	require.Equal(t, "pick an API below, then press v", p.statusMessage)
}

// esc from the count confirm returns to the tree without leaving verifications
// half-done, and the API row stays selected.
func TestVerificationsEscReturnsToTree(t *testing.T) {
	p := newSeedPaneWithWorkspaces([]workspaceRow{{ID: "ws_1", Slug: "acme", Name: "Acme", StripeCustomerID: ""}})
	p.Update(app.KeyMsg{Type: app.KeyRight}) //nolint:exhaustruct
	p.Dispatch(apiChoicesMsg{workspaceID: "ws_1", choices: []namedChoice{{ID: "api_abc", Label: "primary"}}, err: nil})
	p.Update(app.KeyMsg{Type: app.KeyDown}) //nolint:exhaustruct
	p.Update(keyString("v"))
	require.NotNil(t, p.seedConfirm)

	p.Update(app.KeyMsg{Type: app.KeyEsc}) //nolint:exhaustruct
	require.Nil(t, p.seedConfirm)
	row, ok := p.currentRow()
	require.True(t, ok)
	require.Equal(t, rowAPI, row.kind)
}

// Left arrow collapses an expanded workspace.
func TestLeftCollapsesWorkspace(t *testing.T) {
	p := newSeedPaneWithWorkspaces([]workspaceRow{{ID: "ws_1", Slug: "acme", Name: "Acme", StripeCustomerID: ""}})
	p.Update(app.KeyMsg{Type: app.KeyRight}) //nolint:exhaustruct
	require.True(t, p.expanded["ws_1"])
	p.Update(app.KeyMsg{Type: app.KeyLeft}) //nolint:exhaustruct
	require.False(t, p.expanded["ws_1"])
}

func TestFormatCount(t *testing.T) {
	require.Equal(t, "100", formatCount(100))
	require.Equal(t, "1k", formatCount(1000))
	require.Equal(t, "100k", formatCount(100000))
	require.Equal(t, "1M", formatCount(1000000))
}

// verificationsArgsFor mirrors the exec args built in updateSeedConfirm so the
// test asserts the exact command without shelling out.
func verificationsArgsFor(c *seedActionConfirm) []string {
	return []string{
		"dev", "seed", "verifications",
		"--api-id=" + c.apiID,
		"--num-verifications=" + strconv.Itoa(verifCountOptions[c.countIdx]),
	}
}
