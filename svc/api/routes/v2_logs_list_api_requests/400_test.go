package handler_test

import (
	"encoding/base64"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

func pathFilter(f openapi.LogTextFilter) handler.Request {
	return handler.Request{Filters: &openapi.V2LogsListApiRequestsFilters{Path: &f}}
}

// TestListApiRequestsBadRequest covers both layers. The OpenAPI middleware
// rejects bounds, list sizes, unknown fields and operators, and unknown
// methods. The handler rejects a cursor it did not issue. The map bodies hold
// what the typed request cannot express
func TestListApiRequestsBadRequest(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	tests := []struct {
		name string
		body any
	}{
		{name: "limit zero", body: handler.Request{Limit: new(0)}},
		{name: "limit over 100", body: handler.Request{Limit: new(101)}},
		{name: "empty value list", body: pathFilter(openapi.LogTextFilter{Is: &[]string{}})},
		{name: "empty value", body: pathFilter(openapi.LogTextFilter{Contains: &[]string{""}})},
		{name: "value too long", body: pathFilter(openapi.LogTextFilter{Is: &[]string{strings.Repeat("a", 1025)}})},
		{name: "too many values", body: pathFilter(openapi.LogTextFilter{Is: new(slices.Repeat([]string{"/v2/keys."}, 11))})},
		{name: "too many exact values", body: handler.Request{Filters: &openapi.V2LogsListApiRequestsFilters{RequestId: &openapi.LogExactFilter{Is: new(slices.Repeat([]string{"req_KEBAP"}, 26))}}}},
		{name: "unknown method", body: handler.Request{Filters: &openapi.V2LogsListApiRequestsFilters{Method: &openapi.LogMethodFilter{Is: &[]openapi.LogHttpMethod{"BREW"}}}}},
		{name: "repeated method", body: handler.Request{Filters: &openapi.V2LogsListApiRequestsFilters{Method: &openapi.LogMethodFilter{Is: &[]openapi.LogHttpMethod{"GET", "GET"}}}}},
		{name: "negative time", body: handler.Request{Filters: &openapi.V2LogsListApiRequestsFilters{Time: &openapi.LogTimeFilter{Gte: new(int64(-1))}}}},
		{name: "unknown field", body: map[string]any{"filters": map[string]any{"region": map[string]any{"is": []string{"us-east-1"}}}}},
		{name: "unknown operator", body: map[string]any{"filters": map[string]any{"path": map[string]any{"like": []string{"/v2/%"}}}}},
		{name: "operator not in time set", body: map[string]any{"filters": map[string]any{"time": map[string]any{"is": []int64{1}}}}},
		{name: "cursor not base64", body: handler.Request{Cursor: new("KEBAP!")}},
		{name: "cursor not json", body: handler.Request{Cursor: new(base64.RawURLEncoding.EncodeToString([]byte("KEBAP")))}},
		{name: "cursor without time", body: handler.Request{Cursor: new(base64.RawURLEncoding.EncodeToString([]byte("{}")))}},
		{name: "cursor without request id", body: map[string]any{"cursor": base64.RawURLEncoding.EncodeToString([]byte(`{"t":1792021850123}`))}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[any, openapi.BadRequestErrorResponse](h, route, testutil.RootKeyHeaders(rootKey), tt.body)

			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}
}
