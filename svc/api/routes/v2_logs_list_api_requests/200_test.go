package handler_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_list_api_requests"
)

func requestIDs(rows []openapi.ApiRequestLog) []string {
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.RequestId
	}

	return ids
}

// TestListApiRequestsReturnsOwnRowsNewestFirst guarantees an empty request
// lists every row of the caller's workspace, newest first, with every field,
// and never a row of another workspace
func TestListApiRequestsReturnsOwnRowsNewestFirst(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	otherWorkspace := h.CreateWorkspace()
	now := h.Clock.Now().UnixMilli()

	rows := insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 2000},
		schema.ApiRequest{
			WorkspaceID:     workspaceID,
			Time:            now - 1000,
			Host:            "api.unkey.com",
			Method:          "POST",
			Path:            "/v2/keys.updateKey",
			RequestHeaders:  []string{"Content-Type: application/json"},
			RequestBody:     `{"name":"KEBAP"}`,
			ResponseStatus:  404,
			ResponseHeaders: []string{"Content-Type: application/json"},
			ResponseBody:    `{"error":{"status":404}}`,
			Error:           "key not found",
			ServiceLatency:  38,
		},
	)
	insertRequests(t, h, schema.ApiRequest{WorkspaceID: otherWorkspace.ID, Time: now})

	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, auth(rootKey), handler.Request{})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{rows[1].RequestID, rows[0].RequestID}, requestIDs(res.Body.Data))
	require.False(t, res.Body.Pagination.HasMore)
	require.Nil(t, res.Body.Pagination.Cursor)
	require.Equal(t, openapi.ApiRequestLog{
		RequestId:       rows[1].RequestID,
		Time:            now - 1000,
		Host:            "api.unkey.com",
		Method:          "POST",
		Path:            "/v2/keys.updateKey",
		Status:          404,
		LatencyMs:       38,
		Error:           "key not found",
		RequestHeaders:  []string{"Content-Type: application/json"},
		RequestBody:     `{"name":"KEBAP"}`,
		ResponseHeaders: []string{"Content-Type: application/json"},
		ResponseBody:    `{"error":{"status":404}}`,
	}, res.Body.Data[0])
}

// TestListApiRequestsFilters guarantees each filter field reaches the query
// with the combine rule the spec documents
func TestListApiRequestsFilters(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	now := h.Clock.Now().UnixMilli()

	rows := insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 5000, Path: "/v2/keys.createKey", ResponseStatus: 400},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 4000, Path: "/v2/keys.verifyKey", ResponseStatus: 401},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 3000, Path: "/v2/keys.updateKey", ResponseStatus: 429},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 2000, Path: "/v2/apis.listKeys", ResponseStatus: 500, Method: "GET"},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 1000, Path: "/v2/keys.deleteKey", ResponseStatus: 200, Host: "api.unkey.dev"},
	)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	tests := []struct {
		name    string
		filters openapi.V2LogsListApiRequestsFilters
		want    []string
	}{
		{
			name: "path, status range and isNot",
			filters: openapi.V2LogsListApiRequestsFilters{
				Path:   &openapi.LogTextFilter{StartsWith: &[]string{"/v2/keys."}, NotContains: &[]string{"verifyKey"}},
				Status: &openapi.LogIntFilter{Gte: new(int64(400)), Lt: new(int64(600)), IsNot: &[]int64{429}},
			},
			want: []string{rows[0].RequestID},
		},
		{
			name:    "method",
			filters: openapi.V2LogsListApiRequestsFilters{Method: &openapi.LogMethodFilter{Is: &[]openapi.LogHttpMethod{"GET", "PUT"}}},
			want:    []string{rows[3].RequestID},
		},
		{
			name:    "host isNot",
			filters: openapi.V2LogsListApiRequestsFilters{Host: &openapi.LogIdFilter{IsNot: &[]string{"api.unkey.com"}}},
			want:    []string{rows[4].RequestID},
		},
		{
			name:    "request id",
			filters: openapi.V2LogsListApiRequestsFilters{RequestId: &openapi.LogIdFilter{Is: &[]string{rows[2].RequestID}}},
			want:    []string{rows[2].RequestID},
		},
		{
			name:    "time window",
			filters: openapi.V2LogsListApiRequestsFilters{Time: &openapi.LogTimeFilter{Gt: new(now - 5000), Lte: new(now - 3000)}},
			want:    []string{rows[2].RequestID, rows[1].RequestID},
		},
		{
			name:    "no match",
			filters: openapi.V2LogsListApiRequestsFilters{RequestId: &openapi.LogIdFilter{Is: &[]string{uid.New(uid.RequestPrefix)}}},
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, auth(rootKey), handler.Request{Filters: &tt.filters})

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tt.want, requestIDs(res.Body.Data))
		})
	}
}

// TestListApiRequestsPaginates guarantees the cursor walks every row once with
// the same filters, and the last page has no cursor
func TestListApiRequestsPaginates(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	now := h.Clock.Now().UnixMilli()

	rows := insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 5000},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 4000},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 3000},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 2000},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 1000},
	)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	var seen []string
	var cursor *string
	for range 3 {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, auth(rootKey), handler.Request{Limit: new(2), Cursor: cursor})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		seen = append(seen, requestIDs(res.Body.Data)...)
		cursor = res.Body.Pagination.Cursor
		require.Equal(t, cursor != nil, res.Body.Pagination.HasMore)
	}

	require.Nil(t, cursor)
	require.Equal(t, []string{rows[4].RequestID, rows[3].RequestID, rows[2].RequestID, rows[1].RequestID, rows[0].RequestID}, seen)
}

// TestListApiRequestsPaginatesInsideOneMillisecond guarantees a page that ends
// inside a millisecond does not skip the other rows of that millisecond
func TestListApiRequestsPaginatesInsideOneMillisecond(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	now := h.Clock.Now().UnixMilli()

	rows := insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 1},
	)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	var seen []string
	var cursor *string
	for range 2 {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, auth(rootKey), handler.Request{Limit: new(2), Cursor: cursor})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		seen = append(seen, requestIDs(res.Body.Data)...)
		cursor = res.Body.Pagination.Cursor
	}

	require.Nil(t, cursor)
	want := make([]string, len(rows))
	for i, row := range rows {
		want[i] = row.RequestID
	}
	require.ElementsMatch(t, want, seen)
	require.Equal(t, rows[3].RequestID, seen[3], "the older millisecond comes last")
}

// TestListApiRequestsKeepsRetention guarantees a row older than the
// workspace's log retention never returns, even when the filter asks for it
func TestListApiRequestsKeepsRetention(t *testing.T) {
	h, route, workspaceID := newRoute(t)

	retained := insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: h.Clock.Now().Add(-29 * 24 * time.Hour).UnixMilli()},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: h.Clock.Now().Add(-31 * 24 * time.Hour).UnixMilli()},
	)
	rootKey := h.CreateRootKey(workspaceID, requestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, auth(rootKey), handler.Request{
		Filters: &openapi.V2LogsListApiRequestsFilters{Time: &openapi.LogTimeFilter{Gte: new(int64(0))}},
	})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, []string{retained[0].RequestID}, requestIDs(res.Body.Data))
}
