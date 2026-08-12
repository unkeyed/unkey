package handler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test400_RejectedQueries(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	queries := map[string]string{
		"physical table name":    "SELECT count() FROM default.runtime_logs_raw_v1",
		"unqualified table name": "SELECT count() FROM runtime_logs_raw_v1",
		"another endpoint table": "SELECT count() FROM gateway_requests_v1",
	}

	for name, query := range queries {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 400, res.Status, "query must be rejected: %s", name)
		})
	}
}

func Test400_InternalColumnsAreUnreachable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "kebap served"})

	for _, column := range []string{"platform", "k8s_pod_name", "attributes", "expires_at"} {
		t.Run(column, func(t *testing.T) {
			for _, query := range []string{
				"SELECT " + column + " FROM runtime_logs_v1",
				"SELECT count() FROM runtime_logs_v1 WHERE toString(" + column + ") != ''",
				"SELECT max(toString(" + column + ")) FROM runtime_logs_v1",
				"SELECT message FROM runtime_logs_v1 ORDER BY " + column,
				"SELECT count() FROM (SELECT " + column + " FROM runtime_logs_v1)",
				"WITH probe AS (SELECT " + column + " AS leaked FROM runtime_logs_v1) SELECT count() FROM probe",
			} {
				res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
				require.Equal(t, 400, res.Status, "query must not reach %s: %s", column, query)
			}
		})
	}
}

func Test400_ErrorsDoNotDiscloseInternals(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT platform FROM runtime_logs_v1",
	})
	require.Equal(t, 400, res.Status)

	body := strings.ToLower(res.RawBody)
	for _, secret := range []string{"runtime_logs_raw_v1", "workspace_id =", strings.ToLower(workspaceID)} {
		require.NotContains(t, body, secret, "the error response must not disclose %q", secret)
	}
}
