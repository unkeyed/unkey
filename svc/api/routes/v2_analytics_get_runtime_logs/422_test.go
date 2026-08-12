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
// every query. To use all the memory needs a large table and gives a flaky
// result, thus this test injects the failure.
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

// A memory limit is a property of the query, thus it returns 422 and not 500.
func Test422_ExceedsMaxMemory(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")

	route := &Handler{AnalyticsConnectionManager: resourceLimitManager{}}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT severity, count() AS total FROM runtime_logs_v1 GROUP BY severity",
	})
	require.Equal(t, 422, res.Status)
	require.Contains(t, res.RawBody, "query_memory_limit_exceeded")
}
