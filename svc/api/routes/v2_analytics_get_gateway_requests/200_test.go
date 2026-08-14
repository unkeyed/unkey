package handler

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test200_RawRequests guarantees the wildcard permission reads raw rows through
// the public alias.
func Test200_RawRequests(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	row := bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/kebap", Method: "GET", ResponseStatus: 200})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT path, method, response_status FROM gateway_requests_v1",
		})
		require.Equal(c, 200, res.Status)
		require.Len(c, res.Body.Data, 1)
		require.Equal(c, row.Path, res.Body.Data[0]["path"])
		require.Equal(c, "GET", res.Body.Data[0]["method"])
	}, 30*time.Second, time.Second)
}

// Test200_WorkspaceIsolation guarantees the injected workspace filter hides
// another workspace's traffic even when the query asks for every row.
func Test200_WorkspaceIsolation(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/mine", ResponseStatus: 200})
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: otherWorkspace.ID, Path: "/theirs", ResponseStatus: 200})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT path FROM gateway_requests_v1",
		})
		require.Equal(c, 200, res.Status)
		require.Equal(c, []map[string]any{{"path": "/mine"}}, res.Body.Data)
	}, 30*time.Second, time.Second)
}

// Test200_ScopeColumnsAreQueryable guarantees a caller can narrow results to
// one project, app, or environment with its own filter.
func Test200_ScopeColumnsAreQueryable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	mine := bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/wanted", ResponseStatus: 200})
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/unwanted", ResponseStatus: 200})

	for name, query := range map[string]string{
		"project":     fmt.Sprintf("SELECT path FROM gateway_requests_v1 WHERE project_id = '%s'", mine.ProjectID),
		"app":         fmt.Sprintf("SELECT path FROM gateway_requests_v1 WHERE app_id = '%s'", mine.AppID),
		"environment": fmt.Sprintf("SELECT path FROM gateway_requests_v1 WHERE environment_id = '%s'", mine.EnvironmentID),
	} {
		t.Run(name, func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
				require.Equal(c, 200, res.Status)
				require.Equal(c, []map[string]any{{"path": "/wanted"}}, res.Body.Data)
			}, 30*time.Second, time.Second)
		})
	}
}

// Test200_WorkspaceFilterCannotEscapeWithOr guarantees a caller cannot widen
// the injected workspace filter through operator precedence. The parser wraps
// the caller WHERE in parentheses, so `OR 1=1` must not reach another workspace.
func Test200_WorkspaceFilterCannotEscapeWithOr(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/mine", ResponseStatus: 200})
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: otherWorkspace.ID, Path: "/theirs", ResponseStatus: 200})

	for name, query := range map[string]string{
		"or true":              "SELECT path FROM gateway_requests_v1 WHERE path = '/mine' OR 1=1",
		"or on workspace":      "SELECT path FROM gateway_requests_v1 WHERE workspace_id != '' OR 1=1",
		"union branch":         "SELECT path FROM gateway_requests_v1 WHERE path = '/mine' UNION ALL SELECT path FROM gateway_requests_v1",
		"subquery":             "SELECT path FROM (SELECT path FROM gateway_requests_v1)",
		"aliased source":       "SELECT g.path AS path FROM gateway_requests_v1 AS g WHERE 1=1",
		"cte over the raw set": "WITH every AS (SELECT path FROM gateway_requests_v1) SELECT path FROM every",
	} {
		t.Run(name, func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
				require.Equal(c, 200, res.Status)
				require.NotEmpty(c, res.Body.Data)
				for _, row := range res.Body.Data {
					require.Equal(c, "/mine", row["path"], "the other workspace's row must stay invisible")
				}
			}, 30*time.Second, time.Second)
		})
	}
}

// Test200_RetentionBoundary guarantees a query inside the workspace retention
// window succeeds, including at the exact limit and with no time filter at all.
func Test200_RetentionBoundary(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/kebap", ResponseStatus: 200})

	// The default workspace retention is 30 days. The raw table stores time as
	// unix milliseconds, and every rollup stores DateTime, so each shape is
	// asserted against the table it belongs to.
	for name, query := range map[string]string{
		"raw within retention":    "SELECT count() AS total FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 7 DAY)",
		"raw at exact limit":      "SELECT count() AS total FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 30 DAY)",
		"raw without time filter": "SELECT count() AS total FROM gateway_requests_v1",
		"minute rollup":           "SELECT count() AS total FROM gateway_requests_per_minute_v1 WHERE time >= now() - INTERVAL 7 DAY",
		"hour rollup":             "SELECT count() AS total FROM gateway_requests_per_hour_v1 WHERE time >= now() - INTERVAL 7 DAY",
		"day rollup":              "SELECT count() AS total FROM gateway_requests_per_day_v1 WHERE time >= now() - INTERVAL 7 DAY",
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
		})
	}
}

// Test200_CustomRetentionAllowsLongerRange guarantees a workspace with a longer
// retention setting can query the extra days.
func Test200_CustomRetentionAllowsLongerRange(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90))
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() AS total FROM gateway_requests_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 60 DAY)",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
}

// Test200_AggregateQuery guarantees the documented aggregate shape works
// against the raw table.
func Test200_AggregateQuery(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	for range 3 {
		bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/orders", ResponseStatus: 500, TotalLatency: 42})
	}

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT path, count() AS total FROM gateway_requests_v1 WHERE response_status >= 500 GROUP BY path ORDER BY total DESC LIMIT 10",
		})
		require.Equal(c, 200, res.Status)
		require.Len(c, res.Body.Data, 1)
		require.Equal(c, "/orders", res.Body.Data[0]["path"])
	}, 30*time.Second, time.Second)
}

// Test200_EmptyResultAggregate guarantees an aggregate with no matching row
// answers 200 with null. ClickHouse gives NaN for such a query. JSON has no
// encoding for NaN. Thus the value must become null before the response.
func Test200_EmptyResultAggregate(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/kebap", ResponseStatus: 200, TotalLatency: 120})

	// The window closes before the row exists. Thus the range stays empty after
	// the buffer flushes.
	emptyRange := "time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 2 DAY) AND time < toUnixTimestamp64Milli(now64(3) - INTERVAL 1 DAY)"

	for name, tc := range map[string]struct {
		query    string
		expected any
	}{
		"quantile with no match": {
			query:    "SELECT quantile(0.95)(total_latency) AS value FROM gateway_requests_v1 WHERE path = '/absent'",
			expected: nil,
		},
		"quantile over empty range": {
			query:    "SELECT quantile(0.95)(total_latency) AS value FROM gateway_requests_v1 WHERE " + emptyRange,
			expected: nil,
		},
		"average over empty range": {
			query:    "SELECT avg(total_latency) AS value FROM gateway_requests_v1 WHERE " + emptyRange,
			expected: nil,
		},
		"division over empty range": {
			query:    "SELECT sum(total_latency) / count() AS value FROM gateway_requests_v1 WHERE " + emptyRange,
			expected: nil,
		},
		"division by zero": {
			query:    "SELECT sum(total_latency) / countIf(response_status = 599) AS value FROM gateway_requests_v1",
			expected: nil,
		},
		"merge on rollup with no match": {
			query:    "SELECT quantileTDigestMerge(0.95)(latency_p95) AS value FROM gateway_requests_per_minute_v1 WHERE response_status = 599",
			expected: nil,
		},
		"merge on rollup over empty range": {
			query:    "SELECT quantileTDigestMerge(0.95)(latency_p95) AS value FROM gateway_requests_per_hour_v1 WHERE time >= now() - INTERVAL 2 DAY AND time < now() - INTERVAL 1 DAY",
			expected: nil,
		},
		"nullable non-finite value": {
			query:    "SELECT if(1, total_latency / 0, NULL) AS value FROM gateway_requests_v1",
			expected: nil,
		},
		"array keeps finite values": {
			query:    "SELECT groupArray(total_latency) AS value FROM gateway_requests_v1",
			expected: []any{float64(120)},
		},
		"grouped rows keep finite values": {
			query:    "SELECT path, quantile(0.95)(total_latency) AS value FROM gateway_requests_v1 GROUP BY path",
			expected: float64(120),
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: tc.query})
				require.Equal(c, 200, res.Status, "response: %s", res.RawBody)
				require.Len(c, res.Body.Data, 1)
				require.Equal(c, tc.expected, res.Body.Data[0]["value"])
			}, 30*time.Second, time.Second)
		})
	}

	// The percentile example in docs/product/platform/analytics has three columns
	// and an app filter. Each column must give null when no row matches.
	t.Run("documented percentile query", func(t *testing.T) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: `
			SELECT
			  quantileTDigestMerge(0.5)(latency_p50) AS p50,
			  quantileTDigestMerge(0.95)(latency_p95) AS p95,
			  quantileTDigestMerge(0.99)(latency_p99) AS p99
			FROM gateway_requests_per_hour_v1
			WHERE app_id = 'app_absent'
			  AND time >= now() - INTERVAL 24 HOUR`,
		})
		require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
		require.Equal(t, []map[string]any{{"p50": nil, "p95": nil, "p99": nil}}, res.Body.Data)
	})
}

// Test200_LatencyPercentileFromRollup guarantees the merge function documented
// in the spec reads the aggregate states in a rollup table.
func Test200_LatencyPercentileFromRollup(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	bufferRequest(t, h, schema.FrontlineRequest{WorkspaceID: workspaceID, Path: "/slow", ResponseStatus: 200, TotalLatency: 120})

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT quantileTDigestMerge(0.95)(latency_p95) AS p95 FROM gateway_requests_per_minute_v1",
		})
		require.Equal(c, 200, res.Status)
		require.Len(c, res.Body.Data, 1)
		require.NotNil(c, res.Body.Data[0]["p95"])
	}, 30*time.Second, time.Second)
}
