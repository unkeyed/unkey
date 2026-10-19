package handler_test

import (
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_logs_get_api_requests_timeseries"
)

func sumPoints(points []openapi.ApiRequestsTimeseriesPoint) openapi.ApiRequestsTimeseriesPoint {
	var sum openapi.ApiRequestsTimeseriesPoint
	for _, p := range points {
		sum.Success += p.Success
		sum.ClientError += p.ClientError
		sum.ServerError += p.ServerError
		sum.Total += p.Total
	}

	return sum
}

// requireContiguous asserts one point per bucket of width bucket from the
// bucket of start to the bucket of end - 1, the rule clients rely on to draw
// a chart and to read the bucket width from two points
func requireContiguous(t *testing.T, points []openapi.ApiRequestsTimeseriesPoint, bucket time.Duration, start, end int64) {
	t.Helper()

	bucketMs := bucket.Milliseconds()
	first := start - start%bucketMs
	last := (end - 1) - (end-1)%bucketMs
	require.Len(t, points, int((last-first)/bucketMs)+1)
	for i, p := range points {
		require.Equal(t, first+int64(i)*bucketMs, p.Time, "point %d", i)
	}
}

// TestGetApiRequestsTimeseriesCountsStatusClasses guarantees each status class
// lands in its own field, 1xx and 3xx only in total, buckets without rows are
// zero, and rows of another workspace never count
func TestGetApiRequestsTimeseriesCountsStatusClasses(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	otherWorkspace := h.CreateWorkspace()
	now := h.Clock.Now().UnixMilli()
	rowTime := now - 30*time.Minute.Milliseconds()

	insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: rowTime, ResponseStatus: 200},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: rowTime, ResponseStatus: 201},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: rowTime, ResponseStatus: 302},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: rowTime, ResponseStatus: 404},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: rowTime, ResponseStatus: 503},
	)
	insertRequests(t, h, schema.ApiRequest{WorkspaceID: otherWorkspace.ID, Time: rowTime})
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	start := now - time.Hour.Milliseconds()
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{
		Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &openapi.TimeFilter{Gte: new(start), Lt: new(now)}},
	})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	requireContiguous(t, res.Body.Data.Points, time.Minute, start, now)

	want := openapi.ApiRequestsTimeseriesPoint{
		Time:        rowTime - rowTime%time.Minute.Milliseconds(),
		Success:     2,
		ClientError: 1,
		ServerError: 1,
		Total:       5,
	}
	for _, p := range res.Body.Data.Points {
		if p.Time == want.Time {
			require.Equal(t, want, p)
			continue
		}
		require.Zero(t, p.Total, "bucket %d", p.Time)
	}
}

// TestGetApiRequestsTimeseriesBucketWidth guarantees the bucket width each
// window gets, including the dashboard's time presets, and that a window older
// than a table's TTL moves to a table that still holds it. A window ending in
// lt and the dashboard's gte: now - since, lte: now get the same width
func TestGetApiRequestsTimeseriesBucketWidth(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))
	now := h.Clock.Now().UnixMilli()
	day := 24 * time.Hour

	tests := []struct {
		name   string
		window time.Duration
		ago    time.Duration
		bucket time.Duration
	}{
		{name: "1h", window: time.Hour, bucket: time.Minute},
		{name: "2h", window: 2 * time.Hour, bucket: time.Minute},
		{name: "3h", window: 3 * time.Hour, bucket: 5 * time.Minute},
		{name: "6h", window: 6 * time.Hour, bucket: 5 * time.Minute},
		{name: "12h", window: 12 * time.Hour, bucket: 15 * time.Minute},
		{name: "24h", window: day, bucket: 30 * time.Minute},
		{name: "2d", window: 2 * day, bucket: time.Hour},
		{name: "3d", window: 3 * day, bucket: time.Hour},
		{name: "1w", window: 7 * day, bucket: 2 * time.Hour},
		{name: "2w", window: 14 * day, bucket: 6 * time.Hour},
		{name: "4w", window: 28 * day, bucket: 6 * time.Hour},
		{name: "60d", window: 60 * day, bucket: day},
		{name: "1h ten days ago", window: time.Hour, ago: 10 * day, bucket: time.Hour},
		{name: "1d forty days ago", window: day, ago: 40 * day, bucket: day},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The dashboard's queryTime is always behind the server clock
			end := now - time.Minute.Milliseconds() - tt.ago.Milliseconds()
			start := end - tt.window.Milliseconds()
			for _, bounds := range []struct {
				time openapi.TimeFilter
				end  int64
			}{
				{time: openapi.TimeFilter{Gte: new(start), Lt: new(end)}, end: end},
				{time: openapi.TimeFilter{Gte: new(start), Lte: new(end - 1)}, end: end},
				{time: openapi.TimeFilter{Gte: new(start), Lte: new(end)}, end: end + 1},
			} {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{
					Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &bounds.time},
				})

				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				requireContiguous(t, res.Body.Data.Points, tt.bucket, *bounds.time.Gte, bounds.end)
			}
		})
	}
}

// TestGetApiRequestsTimeseriesDefaultWindow guarantees an empty request covers
// the workspace's log retention, which is 30 days in the harness, up to now.
// An upper bound past now, even the largest int64, still ends at now
func TestGetApiRequestsTimeseriesDefaultWindow(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))
	now := h.Clock.Now().UnixMilli()

	tests := []struct {
		name    string
		filters *openapi.V2LogsGetApiRequestsTimeseriesFilters
	}{
		{name: "no filters", filters: nil},
		{name: "lte at the largest int64", filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &openapi.TimeFilter{Lte: new(int64(math.MaxInt64))}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{Filters: tt.filters})

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			requireContiguous(t, res.Body.Data.Points, 6*time.Hour, now-(30*24*time.Hour).Milliseconds(), now)
		})
	}
}

// TestGetApiRequestsTimeseriesFilters guarantees each filter field reaches the
// aggregate query with the combine rule of logs.listApiRequests
func TestGetApiRequestsTimeseriesFilters(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	now := h.Clock.Now().UnixMilli()

	insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 5000, Path: "/v2/keys.createKey", ResponseStatus: 400},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 4000, Path: "/v2/keys.verifyKey", ResponseStatus: 401},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 3000, Path: "/v2/keys.updateKey", ResponseStatus: 429},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 2000, Path: "/v2/apis.listKeys", ResponseStatus: 500, Method: "GET"},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: now - 1000, Path: "/v2/keys.deleteKey", ResponseStatus: 200, Host: "api.unkey.dev"},
	)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	tests := []struct {
		name    string
		filters openapi.V2LogsGetApiRequestsTimeseriesFilters
		total   int64
	}{
		{name: "none", filters: openapi.V2LogsGetApiRequestsTimeseriesFilters{}, total: 5},
		{
			name: "path, status range and isNot",
			filters: openapi.V2LogsGetApiRequestsTimeseriesFilters{
				Path:   &openapi.TextFilter{StartsWith: &[]string{"/v2/keys."}, NotContains: &[]string{"verifyKey"}},
				Status: &openapi.IntFilter{Gte: new(int64(400)), Lt: new(int64(600)), IsNot: &[]int64{429}},
			},
			total: 1,
		},
		{name: "method", filters: openapi.V2LogsGetApiRequestsTimeseriesFilters{Method: &openapi.MethodFilter{Is: &[]openapi.HttpMethod{"GET", "PUT"}}}, total: 1},
		{name: "host isNot", filters: openapi.V2LogsGetApiRequestsTimeseriesFilters{Host: &openapi.ExactFilter{IsNot: &[]string{"api.unkey.com"}}}, total: 1},
		{name: "path contains is literal", filters: openapi.V2LogsGetApiRequestsTimeseriesFilters{Path: &openapi.TextFilter{Contains: &[]string{"keys_"}}}, total: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filters.Time = &openapi.TimeFilter{Gte: new(now - time.Hour.Milliseconds())}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{Filters: &tt.filters})

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tt.total, sumPoints(res.Body.Data.Points).Total)
		})
	}
}

// TestGetApiRequestsTimeseriesKeepsRetention guarantees a row older than the
// workspace's log retention never counts, even when the window covers it
func TestGetApiRequestsTimeseriesKeepsRetention(t *testing.T) {
	h, route, workspaceID := newRoute(t)

	insertRequests(t, h,
		schema.ApiRequest{WorkspaceID: workspaceID, Time: h.Clock.Now().Add(-29 * 24 * time.Hour).UnixMilli()},
		schema.ApiRequest{WorkspaceID: workspaceID, Time: h.Clock.Now().Add(-31 * 24 * time.Hour).UnixMilli()},
	)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{
		Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &openapi.TimeFilter{Gte: new(int64(0))}},
	})

	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, int64(1), sumPoints(res.Body.Data.Points).Total)
}

// TestGetApiRequestsTimeseriesEmptyWindow guarantees a window with nothing to
// count returns no points instead of an error
func TestGetApiRequestsTimeseriesEmptyWindow(t *testing.T) {
	h, route, workspaceID := newRoute(t)
	rootKey := h.CreateRootKey(workspaceID, apiRequestLogsPermission(workspaceID))
	now := h.Clock.Now().UnixMilli()

	tests := []struct {
		name string
		time openapi.TimeFilter
	}{
		{name: "after now", time: openapi.TimeFilter{Gt: new(now)}},
		{name: "gt at the largest int64", time: openapi.TimeFilter{Gt: new(int64(math.MaxInt64))}},
		{name: "start after end", time: openapi.TimeFilter{Gte: new(now - 1000), Lt: new(now - 2000)}},
		{name: "older than any aggregate", time: openapi.TimeFilter{Lt: new(now - (101 * 24 * time.Hour).Milliseconds())}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, testutil.RootKeyHeaders(rootKey), handler.Request{
				Filters: &openapi.V2LogsGetApiRequestsTimeseriesFilters{Time: &tt.time},
			})

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Empty(t, res.Body.Data.Points)
		})
	}
}
