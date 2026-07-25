package cron_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	rldb "github.com/unkeyed/unkey/internal/services/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
)

// TestRunRatelimitGlobalCountersCleanup_Integration checks the cutoff: rows
// past expires_at are swept, rows still inside their window are not. The
// handler issues its DELETE in bounded batches (PlanetScale rejects a single
// DML statement affecting more than 100,000 rows); the batch boundary itself is
// not exercised here since it would take 25k seeded rows to reach.
func TestRunRatelimitGlobalCountersCleanup_Integration(t *testing.T) {
	h := harness.New(t)
	rl := rldb.New(h.DB.RW(), h.DB.RO())

	now := time.Now()
	expired := seedGlobalCounters(t, h, rl, now.Add(-time.Hour).UnixMilli(), 5)
	live := seedGlobalCounters(t, h, rl, now.Add(time.Hour).UnixMilli(), 2)

	client := hydrav1.NewCronServiceIngressClient(h.Restate, "ratelimit-global-counters-cleanup")
	resp, err := client.RunRatelimitGlobalCountersCleanup().
		Request(h.Ctx, &hydrav1.RunRatelimitGlobalCountersCleanupRequest{})
	require.NoError(t, err)
	require.GreaterOrEqual(t, resp.GetRowsDeleted(), int64(5), "every expired row should be deleted")

	require.Empty(t, globalCountersFor(t, h, rl, expired), "no expired row should survive")
	require.Len(t, globalCountersFor(t, h, rl, live), 2, "rows still inside their window must not be touched")
}

// seedGlobalCounters inserts count rows sharing one fresh workspace id, all
// with the same expires_at. Rows differ only by sequence, which
// unique_window_region covers, so every row is distinct. Returns the workspace
// id so the caller can scope assertions to its own rows.
func seedGlobalCounters(
	t *testing.T,
	h *harness.Harness,
	rl *rldb.Database,
	expiresAtMs int64,
	count int,
) string {
	t.Helper()
	workspaceID := uid.New("ws")

	rows := make([]rldb.UpsertRatelimitGlobalCountersParams, 0, count)
	for i := range count {
		rows = append(rows, rldb.UpsertRatelimitGlobalCountersParams{
			WorkspaceID: workspaceID,
			Namespace:   "cleanup-test",
			Identifier:  "user-1",
			DurationMs:  60_000,
			Sequence:    int64(i),
			Region:      "test-region",
			Count:       1,
			ExpiresAt:   uint64(expiresAtMs),
			UpdatedAt:   uint64(expiresAtMs),
		})
	}
	require.NoError(t, rl.BulkUpsertGlobalCounters(h.Ctx, rows))
	return workspaceID
}

func globalCountersFor(
	t *testing.T,
	h *harness.Harness,
	rl *rldb.Database,
	workspaceID string,
) []rldb.GlobalCountersListAllRow {
	t.Helper()
	all, err := rl.RO().GlobalCountersListAll(h.Ctx)
	require.NoError(t, err)

	mine := make([]rldb.GlobalCountersListAllRow, 0, len(all))
	for _, row := range all {
		if row.WorkspaceID == workspaceID {
			mine = append(mine, row)
		}
	}
	return mine
}
