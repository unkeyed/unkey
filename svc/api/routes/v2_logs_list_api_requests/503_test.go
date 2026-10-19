package handler_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

type unreachableClickHouse struct {
	clickhouse.ClickHouse
}

func (unreachableClickHouse) ListApiRequests(context.Context, clickhouse.ListApiRequestsRequest) (clickhouse.ApiRequestsPage, error) {
	return clickhouse.ApiRequestsPage{}, errors.New("dial tcp 10.0.0.9:9000: connect: connection refused")
}

// TestListApiRequestsLogStoreUnavailable guarantees an unreachable log store is
// a 503, which tells the caller to retry, and not a 400 that blames the request
func TestListApiRequestsLogStoreUnavailable(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	route.ClickHouse = unreachableClickHouse{ClickHouse: h.ClickHouse}
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, openapi.ServiceUnavailableErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{})

	require.Equal(t, http.StatusServiceUnavailable, res.Status, "expected 503, received: %s", res.RawBody)
}
