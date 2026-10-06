package caches

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestWorkspaceByOrgIDOpDoesNotCacheMisses(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock()
	store, err := cache.New(cache.Config[string, string]{
		MaxSize:  10,
		Fresh:    time.Minute,
		Stale:    24 * time.Hour,
		Resource: "workspace_by_org_id",
		Clock:    clk,
	})
	require.NoError(t, err)

	calls := 0
	refresh := func(context.Context) (string, error) {
		calls++
		if calls == 1 {
			return "", sql.ErrNoRows
		}
		return "ws_created", nil
	}

	_, hit, err := store.SWR(context.Background(), "org_new", refresh, WorkspaceByOrgIDOp)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.Equal(t, cache.Miss, hit)

	value, hit, err := store.SWR(context.Background(), "org_new", refresh, WorkspaceByOrgIDOp)
	require.NoError(t, err)
	require.Equal(t, "ws_created", value)
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, 2, calls)
}

func TestWorkspaceByOrgIDOpCachesHitsAndIgnoresOtherErrors(t *testing.T) {
	t.Parallel()

	require.Equal(t, cache.Noop, WorkspaceByOrgIDOp(sql.ErrNoRows))
	require.Equal(t, cache.WriteValue, WorkspaceByOrgIDOp(nil))
	require.Equal(t, cache.Noop, WorkspaceByOrgIDOp(errors.New("database unavailable")))
}
