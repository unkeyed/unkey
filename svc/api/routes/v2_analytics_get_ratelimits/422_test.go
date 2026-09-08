package handler

import (
	"context"
	"fmt"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

type resourceLimitConnection struct {
	clickhouse.ClickHouse
}

func (resourceLimitConnection) QueryToMaps(context.Context, string, ...any) ([]map[string]any, error) {
	return nil, &ch.Exception{Code: 241, Name: "MEMORY_LIMIT_EXCEEDED", Message: "memory limit exceeded"}
}

type resourceLimitManager struct{}

func (resourceLimitManager) GetConnection(context.Context, string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	return resourceLimitConnection{}, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseMaxQueryResultRows: 100,
		QuotaLogsRetentionDays:       30,
	}, nil
}

// Test422_ClickHouseResourceLimit guarantees a per-query ClickHouse resource
// limit maps to the public unprocessable-query response.
func Test422_ClickHouseResourceLimit(t *testing.T) {
	h := testutil.NewHarness(t)
	workspace := h.CreateWorkspace()
	id := createNamespace(t, h, workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "ratelimit.*.read_analytics")
	route := &Handler{AnalyticsConnectionManager: resourceLimitManager{}}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: fmt.Sprintf("SELECT identifier, passed, count(*) FROM ratelimits_v1 WHERE namespace_id = '%s' GROUP BY identifier, passed", id)})
	require.Equal(t, 422, res.Status)
	require.Contains(t, res.RawBody, "query_memory_limit_exceeded")
}
