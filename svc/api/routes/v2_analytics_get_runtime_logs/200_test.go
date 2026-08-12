package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func Test200_RawLogs(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, severity: "error", message: "kebap burned"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT severity, message FROM runtime_logs_v1",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, "error", res.Body.Data[0]["severity"])
	require.Equal(t, "kebap burned", res.Body.Data[0]["message"])
}

func Test200_WorkspaceIsolation(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "mine"})
	insertLog(t, h, runtimeLog{workspaceID: otherWorkspace.ID, message: "theirs"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT message FROM runtime_logs_v1",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Equal(t, []map[string]any{{"message": "mine"}}, res.Body.Data)
}

func Test200_ScopeColumnsAreQueryable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	mine := insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "wanted"})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "unwanted"})

	for name, query := range map[string]string{
		"project":     fmt.Sprintf("SELECT message FROM runtime_logs_v1 WHERE project_id = '%s'", mine.projectID),
		"app":         fmt.Sprintf("SELECT message FROM runtime_logs_v1 WHERE app_id = '%s'", mine.appID),
		"environment": fmt.Sprintf("SELECT message FROM runtime_logs_v1 WHERE environment_id = '%s'", mine.environmentID),
		"deployment":  fmt.Sprintf("SELECT message FROM runtime_logs_v1 WHERE deployment_id = '%s'", mine.deploymentID),
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
			require.Equal(t, []map[string]any{{"message": "wanted"}}, res.Body.Data)
		})
	}
}

// The parser puts the WHERE of the caller in parentheses. Thus an OR cannot
// make the injected workspace filter less restrictive.
func Test200_WorkspaceFilterCannotEscapeWithOr(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	otherWorkspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "mine"})
	insertLog(t, h, runtimeLog{workspaceID: otherWorkspace.ID, message: "theirs"})

	for name, query := range map[string]string{
		"or true":              "SELECT message FROM runtime_logs_v1 WHERE message = 'mine' OR 1=1",
		"or on workspace":      "SELECT message FROM runtime_logs_v1 WHERE workspace_id != '' OR 1=1",
		"union branch":         "SELECT message FROM runtime_logs_v1 WHERE message = 'mine' UNION ALL SELECT message FROM runtime_logs_v1",
		"subquery":             "SELECT message FROM (SELECT message FROM runtime_logs_v1)",
		"aliased source":       "SELECT r.message AS message FROM runtime_logs_v1 AS r WHERE 1=1",
		"cte over the raw set": "WITH every AS (SELECT message FROM runtime_logs_v1) SELECT message FROM every",
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
			require.NotEmpty(t, res.Body.Data)
			for _, row := range res.Body.Data {
				require.Equal(t, "mine", row["message"], "the log of the other workspace must stay invisible")
			}
		})
	}
}

// LIKE is an operator, thus the function list does not apply to it. The queries
// put the column in lower() because the ngrambf_v1 index uses lower(message).
func Test200_TextSearch(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "upstream TIMEOUT after 30s"})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "healthcheck ok"})

	for name, query := range map[string]string{
		"like":     "SELECT message FROM runtime_logs_v1 WHERE lower(message) LIKE '%timeout%'",
		"not like": "SELECT message FROM runtime_logs_v1 WHERE lower(message) NOT LIKE '%healthcheck%'",
		"prefix":   "SELECT message FROM runtime_logs_v1 WHERE startsWith(lower(message), 'upstream')",
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
			require.Len(t, res.Body.Data, 1)
			require.Equal(t, "upstream TIMEOUT after 30s", res.Body.Data[0]["message"])
		})
	}
}

// The JSON column has no grant. Thus attributes_text is the only route to the
// log attributes.
func Test200_AttributesAreReadable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	insertLog(t, h, runtimeLog{
		workspaceID: workspaceID,
		message:     "order placed",
		attributes:  `{"route":"/orders","user_id":"usr_kebap"}`,
	})

	t.Run("full attribute string", func(t *testing.T) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT attributes_text FROM runtime_logs_v1",
		})
		require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
		require.Len(t, res.Body.Data, 1)
		require.Contains(t, fmt.Sprint(res.Body.Data[0]["attributes_text"]), "usr_kebap")
	})

	// This query is the reason for the grant on JSONExtractString. To select
	// attributes_text and to parse it in the client cannot group the rows by the
	// value of an attribute.
	t.Run("group by an attribute", func(t *testing.T) {
		res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
			Query: "SELECT JSONExtractString(attributes_text, 'route') AS route, count() AS total FROM runtime_logs_v1 GROUP BY route",
		})
		require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, "/orders", res.Body.Data[0]["route"])
	})
}

// The partitions of the table use inserted_at, but a log query filters on time.
// Without a condition on inserted_at, a query reads the marks of all the
// partitions.
func Test200_PartitionPruningFilterIsQueryable(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "kebap served"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT message FROM runtime_logs_v1 " +
			"WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 24 HOUR) " +
			"AND inserted_at >= toUnixTimestamp64Milli(now64(3) - INTERVAL 26 HOUR)",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
}

func Test200_RetentionBoundary(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, message: "kebap served"})

	// The default workspace retention is 30 days. The table stores time as unix
	// milliseconds, thus each bound uses toUnixTimestamp64Milli.
	for name, query := range map[string]string{
		"within retention":    "SELECT count() AS total FROM runtime_logs_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 7 DAY)",
		"at exact limit":      "SELECT count() AS total FROM runtime_logs_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 30 DAY)",
		"without time filter": "SELECT count() AS total FROM runtime_logs_v1",
	} {
		t.Run(name, func(t *testing.T) {
			res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{Query: query})
			require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
		})
	}
}

func Test200_CustomRetentionAllowsLongerRange(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90))
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_analytics")
	route := &Handler{AnalyticsConnectionManager: h.AnalyticsConnectionManager}
	h.Register(route)

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT count() AS total FROM runtime_logs_v1 WHERE time >= toUnixTimestamp64Milli(now64(3) - INTERVAL 60 DAY)",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
}

func Test200_AggregateQuery(t *testing.T) {
	h, route, workspaceID := newRoute(t, true)
	rootKey := h.CreateRootKey(workspaceID, "project.*.read_analytics")

	deployment := insertLog(t, h, runtimeLog{workspaceID: workspaceID, severity: "error", message: "first failure"})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, deploymentID: deployment.deploymentID, severity: "error", message: "second failure"})
	insertLog(t, h, runtimeLog{workspaceID: workspaceID, deploymentID: deployment.deploymentID, severity: "info", message: "started"})

	res := testutil.CallRoute[Request, Response](h, route, auth(rootKey), Request{
		Query: "SELECT severity, count() AS total FROM runtime_logs_v1 GROUP BY severity ORDER BY total DESC",
	})
	require.Equal(t, 200, res.Status, "response: %s", res.RawBody)
	require.Len(t, res.Body.Data, 2)
	require.Equal(t, "error", res.Body.Data[0]["severity"])
}
