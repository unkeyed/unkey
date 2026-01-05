package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func Test400_EmptyQuery(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Empty query should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "invalid_analytics_query")
	require.NotEmpty(t, res.Body.Error.Detail, "Error should have a descriptive message")
}

func Test400_InvalidSQLSyntax(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT * FROM key_verifications_v1 WHERE invalid syntax here",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Invalid SQL syntax should return 400")
	require.NotNil(t, res.Body)
	// Parser should return invalid_analytics_query for SQL syntax errors
	require.Contains(t, res.Body.Error.Type, "invalid_analytics_query",
		"Error type should be invalid_analytics_query")
	require.NotEmpty(t, res.Body.Error.Detail, "Error should show syntax error message")
}

func Test400_UnknownColumn(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT nonexistent_column FROM key_verifications_v1 WHERE time >= now() - INTERVAL 7 DAY",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Unknown column should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "invalid_analytics_query")
	require.Contains(t, res.Body.Error.Detail, "Unknown", "Error should mention unknown column")
}

func Test400_InvalidTable(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "SELECT * FROM system.tables",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Invalid table should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "invalid_analytics_table")
	require.NotEmpty(t, res.Body.Error.Detail, "Error should have a descriptive message")
}

func Test400_NonSelectQuery(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	req := Request{
		Query: "DELETE FROM key_verifications_v1 WHERE time < now()",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Non-SELECT query should return 400")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "invalid_analytics_query_type")
	require.NotEmpty(t, res.Body.Error.Detail, "Error should have a descriptive message")
}

func Test400_QueryBeyond30Days(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Query 60 days (beyond 30-day retention)
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 60 DAY",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Query beyond retention should fail")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention", "Should return query_range_exceeds_retention error")
	require.Contains(t, res.Body.Error.Detail, "30 days", "Error message should mention retention period")
}

func Test400_QueryBeyondCustomRetention90Days(t *testing.T) {
	h := testutil.NewHarness(t)

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90)) // 90-day retention
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		Logger:                     h.Logger,
		DB:                         h.DB,
		Keys:                       h.Keys,
		ClickHouse:                 h.ClickHouse,
		AnalyticsConnectionManager: h.AnalyticsConnectionManager,
		Caches:                     h.Caches,
	}
	h.Register(route)

	headers := http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}

	// Query 120 days (beyond 90-day retention)
	req := Request{
		Query: "SELECT COUNT(*) as count FROM key_verifications_v1 WHERE time >= now() - INTERVAL 120 DAY",
	}

	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, headers, req)
	require.Equal(t, 400, res.Status, "Query beyond custom retention should fail")
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Type, "query_range_exceeds_retention", "Should return query_range_exceeds_retention error")
	require.Contains(t, res.Body.Error.Detail, "90 days", "Error message should mention custom retention period")
}
