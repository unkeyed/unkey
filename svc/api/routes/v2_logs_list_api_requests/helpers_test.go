package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

func newRoute(t *testing.T) (*testutil.Harness, *handler.Handler, string) {
	t.Helper()

	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	route := &handler.Handler{
		ClickHouse:  h.ClickHouse,
		DB:          h.DB,
		LimitsCache: h.Caches.WorkspaceLimits,
		Clock:       h.Clock,
	}
	h.Register(route)

	return h, route, workspace.ID
}

// insertRequests writes rows synchronously, so a test reads them without
// waiting for a buffer flush
func insertRequests(t *testing.T, h *testutil.Harness, rows ...schema.ApiRequest) []schema.ApiRequest {
	t.Helper()

	batch, err := h.ClickHouse.Conn().PrepareBatch(context.Background(), clickhouse.InsertQuery[schema.ApiRequest]())
	require.NoError(t, err)
	for i := range rows {
		if rows[i].RequestID == "" {
			rows[i].RequestID = uid.New(uid.RequestPrefix)
		}
		if rows[i].Time == 0 {
			rows[i].Time = h.Clock.Now().UnixMilli()
		}
		if rows[i].Host == "" {
			rows[i].Host = "api.unkey.com"
		}
		if rows[i].Method == "" {
			rows[i].Method = "POST"
		}
		if rows[i].Path == "" {
			rows[i].Path = "/v2/keys.createKey"
		}
		if rows[i].ResponseStatus == 0 {
			rows[i].ResponseStatus = 200
		}
		if rows[i].QueryParams == nil {
			rows[i].QueryParams = map[string][]string{}
		}
		if rows[i].RequestHeaders == nil {
			rows[i].RequestHeaders = []string{}
		}
		if rows[i].ResponseHeaders == nil {
			rows[i].ResponseHeaders = []string{}
		}
		require.NoError(t, batch.AppendStruct(&rows[i]))
	}
	require.NoError(t, batch.Send())

	return rows
}

func apiRequestLogsPermission(workspaceID string) string {
	return "unkey:v1:" + workspaceID + ":logs/apiRequests#read"
}
