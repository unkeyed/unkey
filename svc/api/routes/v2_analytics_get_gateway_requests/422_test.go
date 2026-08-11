package handler

import (
	"context"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// resourceLimitConnection returns the ClickHouse resource-limit exception for
// every query. Exhausting real memory needs a large table and gives a flaky
// result, so the failure is injected here instead.
type resourceLimitConnection struct {
	clickhouse.ClickHouse
}

func (resourceLimitConnection) QueryToMaps(context.Context, string, ...any) ([]map[string]any, error) {
	return nil, &ch.Exception{Code: 241, Name: "MEMORY_LIMIT_EXCEEDED", Message: "memory limit exceeded"}
}

type resourceLimitManager struct{}

func (resourceLimitManager) GetConnection(context.Context, string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	return resourceLimitConnection{}, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{MaxQueryResultRows: 100},
		Limit:                      db.Limit{LogsRetentionDaysMax: 30},
	}, nil
}

// Test422_ExceedsMaxMemory guarantees a per-query ClickHouse memory limit maps
// to the public unprocessable-query response and not to a 500.
func Test422_ExceedsMaxMemory(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")

	route := &Handler{AnalyticsConnectionManager: resourceLimitManager{}}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT path, count() AS total FROM gateway_requests_v1 GROUP BY path",
	})
	require.Equal(t, 422, res.Status)
	require.Contains(t, res.RawBody, "query_memory_limit_exceeded")
}
