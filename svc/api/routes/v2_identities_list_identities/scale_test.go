package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

const (
	scaleWorkspaceID = "ws_bench_identity_search"
	scaleIdentities  = 1_000_000

	// The LIKE '%...%' filter cannot use an index, so any search returning
	// fewer rows than the limit scans every identity in the workspace. That
	// worst case measures ~3s locally on a workspace with one million
	// identities; the deadline leaves headroom for slower CI runners while
	// still catching order-of-magnitude regressions.
	scaleSearchDeadline = 30 * time.Second
)

// TestSearchAtScale seeds a workspace with one million identities and asserts
// that searches through the full route (auth, RBAC, validation, query,
// serialization) return correct results within scaleSearchDeadline.
func TestSearchAtScale(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{MySQLDiskStorage: true})
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	ctx := context.Background()
	seedScaleIdentities(t, ctx, h)

	rootKey := h.CreateRootKey(scaleWorkspaceID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	testCases := []struct {
		name     string
		search   string
		wantRows int
	}{
		// Paginating without a search term, for comparison
		{"no search", "", 100},
		// Matches exactly one row, still scans the whole workspace
		{"selective hit", "bench_000000042", 1},
		// Matches 10 scattered rows, fewer than the limit, so the scan
		// cannot stop early and reads the whole workspace
		{"partial page hit", "bench_00000004", 10},
		// Matches every row, terminates early once the page is full
		{"broad hit", "user_bench", 100},
		// Matches nothing, scans the whole workspace
		{"miss", "does-not-exist", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := handler.Request{}
			if tc.search != "" {
				req.Search = &tc.search
			}

			start := time.Now()
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			elapsed := time.Since(start)

			require.Equal(t, 200, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)
			require.Len(t, res.Body.Data, tc.wantRows)
			require.Less(t, elapsed, scaleSearchDeadline)
			t.Logf("search %q returned %d rows in %s", tc.search, len(res.Body.Data), elapsed)
		})
	}
}

// seedScaleIdentities inserts one million identities in a single
// INSERT ... SELECT off a recursive CTE. The disk-backed container is reused
// across local runs, so seeding is skipped when the rows already exist.
func seedScaleIdentities(t *testing.T, ctx context.Context, h *testutil.Harness) {
	t.Helper()

	var seeded bool
	err := h.DB.RO().QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM workspaces WHERE id = ?)
		AND (SELECT COUNT(*) FROM identities WHERE workspace_id = ?) = ?`,
		scaleWorkspaceID, scaleWorkspaceID, scaleIdentities,
	).Scan(&seeded)
	require.NoError(t, err)
	if seeded {
		seedScaleQuota(t, ctx, h)
		return
	}

	seedStart := time.Now()

	tx, err := h.DB.RW().Begin(ctx)
	require.NoError(t, err)
	defer func() {
		err := tx.Rollback()
		require.True(t, err == nil || errors.Is(err, sql.ErrTxDone), "unexpected rollback error: %v", err)
	}()

	// Drop leftovers from an interrupted run before reseeding
	_, err = tx.ExecContext(ctx, "DELETE FROM identities WHERE workspace_id = ?", scaleWorkspaceID)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "DELETE FROM workspaces WHERE id = ?", scaleWorkspaceID)
	require.NoError(t, err)

	err = db.Query.InsertWorkspace(ctx, tx, db.InsertWorkspaceParams{
		ID:        scaleWorkspaceID,
		Name:      "Identity Search Scale Test",
		Slug:      scaleWorkspaceID,
		OrgID:     uid.New(uid.OrgPrefix),
		CreatedAt: time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// The recursion depth override applies to the transaction's connection,
	// which is the one running the CTE
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf("SET SESSION cte_max_recursion_depth = %d", scaleIdentities+1),
	)
	require.NoError(t, err)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO identities (id, external_id, workspace_id, environment, deleted, created_at)
		WITH RECURSIVE seq (n) AS (
			SELECT 1
			UNION ALL
			SELECT n + 1 FROM seq WHERE n < ?
		)
		SELECT
			CONCAT('id_bench_', LPAD(n, 9, '0')),
			CONCAT('user_bench_', LPAD(n, 9, '0')),
			?,
			'default',
			false,
			0
		FROM seq`,
		scaleIdentities,
		scaleWorkspaceID,
	)
	require.NoError(t, err)

	require.NoError(t, tx.Commit())
	seedScaleQuota(t, ctx, h)
	t.Logf("seeded %d identities in %s", scaleIdentities, time.Since(seedStart))
}

// seedScaleQuota ensures the scale workspace has a quota row. Without one the
// auth middleware's workspace rate limiting fails open after logging a quota
// lookup error, which is not the path production requests take.
func seedScaleQuota(t *testing.T, ctx context.Context, h *testutil.Harness) {
	t.Helper()

	err := db.Query.UpsertQuota(ctx, h.DB.RW(), db.UpsertQuotaParams{
		WorkspaceID:            scaleWorkspaceID,
		LogsRetentionDays:      30,
		AuditLogsRetentionDays: 30,
		RequestsPerMonth:       1_000_000,
		Team:                   false,
		RatelimitApiLimit:      sql.NullInt32{}, //nolint:exhaustruct
		RatelimitApiDuration:   sql.NullInt32{}, //nolint:exhaustruct
	})
	require.NoError(t, err)
}
