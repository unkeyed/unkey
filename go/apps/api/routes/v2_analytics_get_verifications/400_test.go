package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/pkg/testutil"
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
	require.Contains(t, res.Body.Error.Type, "invalid_input")
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
	// Parser may catch this as invalid_input or invalid_analytics_query depending on when it's detected
	require.True(t,
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/application/invalid_input" ||
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/user/bad_request/invalid_analytics_query",
		"Error type should be invalid_input or invalid_analytics_query")
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
	// Parser may catch this as invalid_input or invalid_analytics_table depending on when it's detected
	require.True(t,
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/application/invalid_input" ||
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/user/bad_request/invalid_analytics_table",
		"Error type should be invalid_input or invalid_analytics_table")
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
	// Parser may catch this as invalid_input or invalid_analytics_query_type depending on when it's detected
	require.True(t,
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/application/invalid_input" ||
		res.Body.Error.Type == "https://unkey.com/docs/errors/unkey/user/bad_request/invalid_analytics_query_type",
		"Error type should be invalid_input or invalid_analytics_query_type")
	require.NotEmpty(t, res.Body.Error.Detail, "Error should have a descriptive message")
}
