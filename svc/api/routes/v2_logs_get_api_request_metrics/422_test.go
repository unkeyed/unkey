package handler_test

import (
	"context"
	"net/http"
	"testing"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_request_metrics"
)

type limitedClickHouse struct {
	clickhouse.ClickHouse
}

func (limitedClickHouse) GetApiRequestMetrics(context.Context, clickhouse.ApiRequestMetricsRequest) ([]clickhouse.ApiRequestMetricsPoint, error) {
	return nil, clickhouse.WrapClickHouseError(&ch.Exception{Code: 159, Name: "DB::Exception", Message: "Timeout exceeded: elapsed 10.1 seconds, maximum: 10"})
}

// TestGetApiRequestMetricsReadLimit guarantees a read limit is a 422 that
// names what the caller can change
func TestGetApiRequestMetricsReadLimit(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	route.ClickHouse = limitedClickHouse{ClickHouse: h.ClickHouse}
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, openapi.UnprocessableEntityErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{})

	require.Equal(t, http.StatusUnprocessableEntity, res.Status, "expected 422, received: %s", res.RawBody)
	require.Equal(t, "The query exceeds a read limit. Send a narrower `filters.time` window or more filters, then try again.", res.Body.Error.Detail)
}
