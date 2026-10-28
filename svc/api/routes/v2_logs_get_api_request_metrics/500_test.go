package handler_test

import (
	"context"
	"net/http"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_request_metrics"
)

// TestGetApiRequestMetricsWithoutLimits guarantees a workspace with no
// limits row never reads without its retention bound, also from the cached miss
func TestGetApiRequestMetricsWithoutLimits(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	require.NoError(t, db.Query.DeleteLimitByWorkspaceID(context.Background(), h.DB.RW(), workspaceID))
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	for range 2 {
		res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{})
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	}
}

type rejectingClickHouse struct {
	clickhouse.ClickHouse
}

func (rejectingClickHouse) GetApiRequestMetrics(context.Context, clickhouse.ApiRequestMetricsRequest) ([]clickhouse.ApiRequestMetricsPoint, error) {
	return nil, fault.Wrap(&ch.Exception{Code: 47, Name: "DB::Exception", Message: "Unknown identifier"}, fault.Internal("failed to read the api request metrics"))
}

// TestGetApiRequestMetricsQueryRejected guarantees ClickHouse rejecting
// the query is a 500, because Unkey writes the query
func TestGetApiRequestMetricsQueryRejected(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	route.ClickHouse = rejectingClickHouse{ClickHouse: h.ClickHouse}
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{})

	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
}
