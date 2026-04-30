package coordinator

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestMakeGroupKey_SameInputsSameKey(t *testing.T) {
	t.Parallel()

	a := MakeGroupKey("ws_1", "proj_1", "env_prod", SourceRuntime)
	b := MakeGroupKey("ws_1", "proj_1", "env_prod", SourceRuntime)
	require.Equal(t, a, b)
}

func TestMakeGroupKey_DifferentSourcesDifferentKeys(t *testing.T) {
	t.Parallel()

	rt := MakeGroupKey("ws_1", "proj_1", "env_prod", SourceRuntime)
	rq := MakeGroupKey("ws_1", "proj_1", "env_prod", SourceRequest)
	require.NotEqual(t, rt, rq)
}

func TestMakeGroupKey_BoundaryCollisionResistance(t *testing.T) {
	t.Parallel()

	// MakeGroupKey("a", "b|c", ...) must NOT collide with
	// MakeGroupKey("a|b", "c", ...). The U+001E separator buys us this;
	// switching to a printable separator (e.g. ":") would silently break
	// it once an ID containing ":" appears.
	a := MakeGroupKey("a", "b|c", "env", SourceRuntime)
	b := MakeGroupKey("a|b", "c", "env", SourceRuntime)
	require.NotEqual(t, a, b)
}

func TestBuildGroups_FansOutSourcesAndEnvs(t *testing.T) {
	t.Parallel()

	rows := []db.ListEnabledLogDrainsRow{
		{
			ID:           "ld_1",
			WorkspaceID:  "ws_1",
			ProjectID:    sql.NullString{String: "proj_1", Valid: true},
			Sources:      json.RawMessage(`["runtime","request"]`),
			Environments: json.RawMessage(`["production","preview"]`),
			Apps:         json.RawMessage(`[]`),
			Filters:      json.RawMessage(`{}`),
		},
	}
	groups, err := BuildGroups(rows)
	require.NoError(t, err)
	// 2 sources x 2 envs = 4 groups for one drain.
	require.Len(t, groups, 4)
	for _, g := range groups {
		require.Len(t, g.Drains, 1)
		require.Equal(t, "ld_1", g.Drains[0].ID)
	}
}

func TestBuildGroups_SharedGroupCollapsesDrains(t *testing.T) {
	t.Parallel()

	// Two drains on the same project's prod runtime stream must collapse
	// into a single Group entry — that is the whole read-amplification
	// fix. Otherwise the coordinator queries CH twice for one stream.
	mk := func(id string) db.ListEnabledLogDrainsRow {
		return db.ListEnabledLogDrainsRow{
			ID:           id,
			WorkspaceID:  "ws_1",
			ProjectID:    sql.NullString{String: "proj_1", Valid: true},
			Sources:      json.RawMessage(`["runtime"]`),
			Environments: json.RawMessage(`["production"]`),
			Apps:         json.RawMessage(`[]`),
			Filters:      json.RawMessage(`{}`),
		}
	}
	groups, err := BuildGroups([]db.ListEnabledLogDrainsRow{mk("ld_a"), mk("ld_b")})
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Drains, 2)
}

func TestBuildGroups_WorkspaceWideStaysAsItsOwnGroup(t *testing.T) {
	t.Parallel()

	// project_id NULL = workspace-wide drain. It must NOT merge with any
	// project-scoped drain even if everything else lines up; the worker
	// has its own membership check against actual CH rows.
	rows := []db.ListEnabledLogDrainsRow{
		{
			ID:           "ld_workspace",
			WorkspaceID:  "ws_1",
			ProjectID:    sql.NullString{Valid: false},
			Sources:      json.RawMessage(`["runtime"]`),
			Environments: json.RawMessage(`["production"]`),
			Apps:         json.RawMessage(`[]`),
			Filters:      json.RawMessage(`{}`),
		},
		{
			ID:           "ld_proj",
			WorkspaceID:  "ws_1",
			ProjectID:    sql.NullString{String: "proj_1", Valid: true},
			Sources:      json.RawMessage(`["runtime"]`),
			Environments: json.RawMessage(`["production"]`),
			Apps:         json.RawMessage(`[]`),
			Filters:      json.RawMessage(`{}`),
		},
	}
	groups, err := BuildGroups(rows)
	require.NoError(t, err)
	require.Len(t, groups, 2)
}
