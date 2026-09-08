package handler

import (
	"context"
	"net/http"
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

// Test422_ExceedsMaxMemory guarantees a per-query ClickHouse memory limit maps
// to the public unprocessable-query response.
func Test422_ExceedsMaxMemory(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
		AnalyticsConnectionManager: resourceLimitManager{},
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT key_id, region, outcome, COUNT(*) as count FROM key_verifications_v1 GROUP BY key_id, region, outcome",
	}

	res := testutil.CallRoute[Request, Response](h, route, headers, req)
	require.Equal(t, 422, res.Status)
	require.Contains(t, res.RawBody, "query_memory_limit_exceeded")
}
