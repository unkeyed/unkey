package handler

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test503_ClickHouseConnectionFailure guarantees a backend connection failure
// is reported as unavailable and not as invalid caller SQL.
func Test503_ClickHouseConnectionFailure(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")
	now := h.Clock.Now().UnixMilli()

	require.NoError(t, db.Query.InsertClickhouseWorkspaceSettings(context.Background(), h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID: workspace.ID, Username: workspace.ID, PasswordEncrypted: "invalid_password", QuotaDurationSeconds: 3600,
		MaxQueriesPerWindow: 1000, MaxExecutionTimePerWindow: 1800, MaxQueryExecutionTime: 30, MaxQueryMemoryBytes: 1_000_000_000,
		MaxQueryResultRows: 10_000_000, CreatedAt: now, UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
	}))

	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() FROM gateway_requests_v1",
	})
	require.Equal(t, 503, res.Status)
}
