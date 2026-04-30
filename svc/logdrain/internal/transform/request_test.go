package transform

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

func TestRequest_BasicMapping(t *testing.T) {
	t.Parallel()

	row := schema.SentinelRequest{
		RequestID:       "req_1",
		Time:            1700000000000,
		WorkspaceID:     "ws_1",
		ProjectID:       "proj_2",
		EnvironmentID:   "env_3",
		DeploymentID:    "dep_5",
		Method:          "GET",
		Path:            "/api/v1/users",
		ResponseStatus:  500,
		TotalLatency:    234,
		InstanceLatency: 200,
		SentinelLatency: 34,
		RequestBody:     "should-not-leak",
		ResponseBody:    "also-should-not-leak",
	}
	rec, ok := Request(row, RequestFilter{})
	require.True(t, ok)
	require.Equal(t, sinks.RecordRequest, rec.Kind)
	require.Equal(t, "GET /api/v1/users 500", rec.Body)
	require.Equal(t, "error", rec.SeverityText)
	require.Equal(t, "GET", rec.Attributes["http.request.method"])
	require.Equal(t, int64(500), rec.Attributes["http.response.status_code"])
	require.Equal(t, int64(234), rec.Attributes["unkey.total_latency_ms"])
	// Bodies must NOT appear when IncludeBodies is false.
	require.NotContains(t, rec.Attributes, "http.request.body")
	require.NotContains(t, rec.Attributes, "http.response.body")
}

func TestRequest_IncludeBodiesOptIn(t *testing.T) {
	t.Parallel()

	row := schema.SentinelRequest{
		Path:           "/x",
		Method:         "POST",
		ResponseStatus: 200,
		RequestBody:    "{\"hello\":\"world\"}",
		ResponseBody:   "ok",
	}
	rec, ok := Request(row, RequestFilter{IncludeBodies: true})
	require.True(t, ok)
	require.Equal(t, "{\"hello\":\"world\"}", rec.Attributes["http.request.body"])
	require.Equal(t, "ok", rec.Attributes["http.response.body"])
}

func TestRequest_StatusMatchers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		matcher string
		status  int32
		want    bool
	}{
		{"200", 200, true},
		{"200", 201, false},
		{">=400", 200, false},
		{">=400", 400, true},
		{">=400", 500, true},
		{"<300", 200, true},
		{"<300", 300, false},
		{"5xx", 500, true},
		{"5xx", 599, true},
		{"5xx", 499, false},
		{"4xx", 401, true},
		{"4xx", 500, false},
	}
	for _, c := range cases {
		row := schema.SentinelRequest{ResponseStatus: c.status, Method: "GET", Path: "/x"}
		_, ok := Request(row, RequestFilter{StatusMatchers: []string{c.matcher}})
		require.Equal(t, c.want, ok, "matcher=%q status=%d", c.matcher, c.status)
	}
}

func TestRequest_StatusMatchersOR(t *testing.T) {
	t.Parallel()

	// Multiple matchers OR together: "give me 5xx OR 404".
	row := schema.SentinelRequest{ResponseStatus: 404, Method: "GET", Path: "/x"}
	_, ok := Request(row, RequestFilter{StatusMatchers: []string{"5xx", "404"}})
	require.True(t, ok)

	row.ResponseStatus = 401
	_, ok = Request(row, RequestFilter{StatusMatchers: []string{"5xx", "404"}})
	require.False(t, ok)
}

func TestRequest_ExcludePaths(t *testing.T) {
	t.Parallel()

	row := schema.SentinelRequest{Method: "GET", Path: "/healthz", ResponseStatus: 200}
	_, ok := Request(row, RequestFilter{ExcludePaths: []string{"/healthz"}})
	require.False(t, ok)

	// Prefix match: /healthz/live also drops.
	row.Path = "/healthz/live"
	_, ok = Request(row, RequestFilter{ExcludePaths: []string{"/healthz"}})
	require.False(t, ok)

	// Other paths pass through.
	row.Path = "/api/users"
	_, ok = Request(row, RequestFilter{ExcludePaths: []string{"/healthz"}})
	require.True(t, ok)
}
