package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

// Test400_RejectedQueries guarantees the parser refuses every query shape that
// the endpoint does not support, and that ClickHouse errors come back as caller
// errors rather than as a 500.
func Test400_RejectedQueries(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_gateway_requests")

	queries := map[string]string{
		"empty query":            "",
		"invalid syntax":         "SELECT FROM WHERE",
		"non select":             "DROP TABLE default.frontline_requests_raw_v1",
		"introspection":          "DESCRIBE TABLE gateway_requests_v1",
		"multiple statements":    "SELECT count() FROM gateway_requests_v1; SELECT count() FROM gateway_requests_v1",
		"physical table name":    "SELECT count() FROM default.frontline_requests_raw_v1",
		"ungranted 5m rollup":    "SELECT count() FROM gateway_requests_per_5m_v1",
		"another endpoint table": "SELECT count() FROM key_verifications_v1",
		"system table":           "SELECT count() FROM system.tables",
		"unknown column":         "SELECT no_such_column FROM gateway_requests_v1",
		"blocked function":       "SELECT file('/etc/passwd') FROM gateway_requests_v1",
		"settings clause":        "SELECT count() FROM gateway_requests_v1 SETTINGS max_execution_time = 100",
		"over length query":      "SELECT count() FROM gateway_requests_v1 WHERE path = '" + strings.Repeat("k", 17*1024) + "'",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 400, res.Status, "query must be rejected: %s", name)
		})
	}
}

// Test400_InternalColumnsAreUnreachable guarantees the ClickHouse column grant
// keeps the Unkey infrastructure columns out of reach. The query parser has no
// column allow-list, so this is the only control that stops these reads.
func Test400_InternalColumnsAreUnreachable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_gateway_requests")

	for _, column := range []string{"instance_address", "frontline_id", "platform"} {
		t.Run(column, func(t *testing.T) {
			for _, query := range []string{
				"SELECT " + column + " FROM gateway_requests_v1",
				"SELECT count() FROM gateway_requests_v1 WHERE " + column + " != ''",
				"SELECT max(" + column + ") FROM gateway_requests_v1",
				"SELECT count() FROM (SELECT " + column + " FROM gateway_requests_v1)",
			} {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
				require.Equal(t, 400, res.Status, "query must not reach %s: %s", column, query)
			}
		})
	}
}

// Test400_MessagesNameTheFix guarantees the two ClickHouse failures that a
// caller reaches on these tables give an actionable message instead of the
// generic one.
//
// SELECT * fails because ClickHouse expands the star to every physical column,
// which includes the columns outside the grant. A percentile column fails
// because the driver cannot decode an aggregate state.
func Test400_MessagesNameTheFix(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_gateway_requests")

	for name, testCase := range map[string]struct {
		query   string
		message string
	}{
		"select star on the raw table": {
			query:   "SELECT * FROM gateway_requests_v1",
			message: "Select only the documented columns instead of *",
		},
		"aggregate state column": {
			query:   "SELECT latency_p95 FROM gateway_requests_per_hour_v1 WHERE time >= now() - INTERVAL 1 DAY",
			message: "Use quantileTDigestMerge to read a percentile from it",
		},
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: testCase.query})
			require.Equal(t, 400, res.Status, "response: %s", res.RawBody)
			require.Contains(t, res.RawBody, testCase.message)
		})
	}
}

// Test400_ErrorsDoNotDiscloseInternals guarantees a rejection never returns the
// rewritten SQL, the physical table names, or the injected workspace filter.
func Test400_ErrorsDoNotDiscloseInternals(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_gateway_requests")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT instance_address FROM gateway_requests_v1",
	})
	require.Equal(t, 400, res.Status)

	body := strings.ToLower(res.RawBody)
	for _, secret := range []string{"frontline_requests_raw_v1", "workspace_id =", strings.ToLower(workspaceID)} {
		require.NotContains(t, body, secret, "the error response must not disclose %q", secret)
	}
}
