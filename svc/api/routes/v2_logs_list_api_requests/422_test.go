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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

type limitedClickHouse struct {
	clickhouse.ClickHouse
	exception *ch.Exception
}

func (l limitedClickHouse) ListApiRequests(context.Context, clickhouse.ListApiRequestsRequest) (clickhouse.ApiRequestsPage, error) {
	return clickhouse.ApiRequestsPage{}, clickhouse.WrapClickHouseError(l.exception)
}

// TestListApiRequestsReadLimit guarantees a read limit is a 422 that names
// what the caller can change, not the generic advice to simplify a query
func TestListApiRequestsReadLimit(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	tests := []struct {
		name      string
		exception *ch.Exception
	}{
		{name: "timeout", exception: &ch.Exception{Code: 159, Name: "DB::Exception", Message: "Timeout exceeded: elapsed 10.1 seconds, maximum: 10"}},
		{name: "result too large", exception: &ch.Exception{Code: 396, Name: "DB::Exception", Message: "Limit for result exceeded, max bytes: 4.00 MiB"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route.ClickHouse = limitedClickHouse{ClickHouse: h.ClickHouse, exception: tt.exception}
			res := testutil.CallRoute[handler.Request, openapi.UnprocessableEntityErrorResponse](h, route, auth(rootKey), handler.Request{})

			require.Equal(t, http.StatusUnprocessableEntity, res.Status, "expected 422, received: %s", res.RawBody)
			require.Equal(t, "The query exceeds a read limit. Send a lower `limit`, a narrower `filters.time` window, or more filters, then try again.", res.Body.Error.Detail)
		})
	}
}
