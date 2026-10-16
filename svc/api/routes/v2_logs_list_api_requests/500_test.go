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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

// TestListApiRequestsWithoutLimits guarantees a workspace with no limits row
// never lists without its retention bound. The second call reads the cached
// miss, which must fail the same way as the first
func TestListApiRequestsWithoutLimits(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	require.NoError(t, db.Query.DeleteLimitByWorkspaceID(context.Background(), h.DB.RW(), workspaceID))
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	for range 2 {
		res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, auth(rootKey), handler.Request{})
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
	}
}

type rejectingClickHouse struct {
	clickhouse.ClickHouse
}

func (rejectingClickHouse) ListApiRequests(context.Context, clickhouse.ListApiRequestsRequest) (clickhouse.ApiRequestsPage, error) {
	return clickhouse.ApiRequestsPage{}, fault.Wrap(&ch.Exception{Code: 47, Name: "DB::Exception", Message: "Unknown identifier"}, fault.Internal("failed to list api requests"))
}

// TestListApiRequestsQueryRejected guarantees ClickHouse rejecting the query is
// a 500: the query is written by Unkey, so it is neither the caller's fault
// nor an outage to retry through
func TestListApiRequestsQueryRejected(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	route.ClickHouse = rejectingClickHouse{ClickHouse: h.ClickHouse}
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, openapi.InternalServerErrorResponse](h, route, auth(rootKey), handler.Request{})

	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
}
