package gdprerasure_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
)

// erase submits a GDPR erasure under a fresh request reference and waits for
// its result, as an operator does over Restate ingress.
func erase(t *testing.T, h *harness.Harness, req *hydrav1.GdprErasureRequest) (*hydrav1.GdprErasureResponse, error) {
	t.Helper()
	invocation, err := hydrav1.NewGdprErasureWorkflowIngressClient(h.Restate, uid.New(uid.TestPrefix)).Submit(h.Ctx, req)
	require.NoError(t, err)
	return invocation.Attach(h.Ctx)
}

// requireNothingLeft asserts that a repeated erasure found nothing to do in any
// step.
func requireNothingLeft(t *testing.T, res *hydrav1.GdprErasureResponse) {
	t.Helper()
	require.NotEmpty(t, res.GetSteps())
	for _, step := range res.GetSteps() {
		require.Zero(t, step.GetRowsAffected(), "step %q still found rows", step.GetName())
	}
}

// clickhouseCounts counts the rows matching where in every ClickHouse table
// that stores column. The table list comes from ClickHouse's system tables,
// not from the erasure registry, so a table the registry misses still shows up.
func clickhouseCounts(t *testing.T, h *harness.Harness, column, where string, args ...any) map[string]uint64 {
	t.Helper()

	// Reads system tables, which have no reader function.
	tables, err := h.ClickHouseConn.Query(h.Ctx, `
		SELECT c.table FROM system.columns c
		JOIN system.tables s ON s.database = c.database AND s.name = c.table
		WHERE c.database = 'default' AND c.name = ? AND s.engine LIKE '%MergeTree'`, column)
	require.NoError(t, err)
	var names []string
	for tables.Next() {
		var name string
		require.NoError(t, tables.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, tables.Err())
	require.NoError(t, tables.Close())
	require.NotEmpty(t, names)

	counts := make(map[string]uint64)
	for _, name := range names {
		var count uint64
		require.NoError(t, h.ClickHouseConn.QueryRow(h.Ctx, "SELECT count() FROM default."+name+" WHERE "+where, args...).Scan(&count))
		if count > 0 {
			counts[name] = count
		}
	}
	return counts
}
