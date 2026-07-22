package handler

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test503_ClickHouseConnectionFailure guarantees backend connection failures
// are reported as unavailable rather than invalid caller SQL.
func Test503_ClickHouseConnectionFailure(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	id := createNamespace(t, h, workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "ratelimit.*.read_analytics")
	require.NoError(t, db.Query.UpsertQuota(context.Background(), h.DB.RW(), db.UpsertQuotaParams{WorkspaceID: workspace.ID, LogsRetentionDays: 30, AuditLogsRetentionDays: 30, RequestsPerMonth: 1_000_000}))
	now := h.Clock.Now().UnixMilli()
	require.NoError(t, db.Query.InsertClickhouseWorkspaceSettings(context.Background(), h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID: workspace.ID, Username: workspace.ID, PasswordEncrypted: "invalid_password", QuotaDurationSeconds: 3600,
		MaxQueriesPerWindow: 1000, MaxExecutionTimePerWindow: 1800, MaxQueryExecutionTime: 30, MaxQueryMemoryBytes: 1_000_000_000,
		MaxQueryResultRows: 10_000_000, CreatedAt: now, UpdatedAt: sql.NullInt64{Int64: now, Valid: true},
	}))
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)
	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT count(*) FROM ratelimits_v1 WHERE namespace_id = '%s'", id)})
	require.Equal(t, 503, res.Status)
}
