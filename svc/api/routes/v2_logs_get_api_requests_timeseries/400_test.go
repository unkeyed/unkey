package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_requests_timeseries"
)

// TestGetApiRequestsTimeseriesBadRequest covers what the OpenAPI middleware
// rejects. The map bodies hold what the typed request cannot express
func TestGetApiRequestsTimeseriesBadRequest(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	tests := []struct {
		name string
		body any
	}{
		{name: "request id filter", body: map[string]any{"filters": map[string]any{"requestId": map[string]any{"is": []string{"req_1234abcd"}}}}},
		{name: "empty value list", body: handler.Request{Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Path: &openapi.TextFilter{Is: &[]string{}}}}},
		{name: "unknown method", body: handler.Request{Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Method: &openapi.MethodFilter{Is: &[]openapi.HttpMethod{"BREW"}}}}},
		{name: "negative time", body: handler.Request{Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &openapi.TimeFilter{Gte: new(int64(-1))}}}},
		{name: "unknown operator", body: map[string]any{"filters": map[string]any{"path": map[string]any{"like": []string{"/v2/%"}}}}},
		{name: "granularity", body: map[string]any{"granularity": "hour"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[any, openapi.BadRequestErrorResponse](h, route, auth(rootKey), tt.body)

			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
