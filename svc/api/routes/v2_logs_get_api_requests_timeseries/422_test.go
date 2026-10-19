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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_requests_timeseries"
)

type limitedClickHouse struct {
	clickhouse.ClickHouse
}

func (limitedClickHouse) GetApiRequestsTimeseries(context.Context, clickhouse.ApiRequestsTimeseriesRequest) (clickhouse.ApiRequestsTimeseries, error) {
	return clickhouse.ApiRequestsTimeseries{}, clickhouse.WrapClickHouseError(&ch.Exception{Code: 159, Name: "DB::Exception", Message: "Timeout exceeded: elapsed 10.1 seconds, maximum: 10"})
}

// TestGetApiRequestsTimeseriesReadLimit guarantees a read limit is a 422 that
// names what the caller can change
func TestGetApiRequestsTimeseriesReadLimit(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	route.ClickHouse = limitedClickHouse{ClickHouse: h.ClickHouse}
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, openapi.UnprocessableEntityErrorResponse](h, route, auth(rootKey), handler.Request{})

	require.Equal(t, http.StatusUnprocessableEntity, res.Status, "expected 422, received: %s", res.RawBody)
	require.Equal(t, "The query exceeds a read limit. Send a narrower `filters.time` window or more filters, then try again.", res.Body.Error.Detail)
}
