package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type countingCache[K comparable, V any] struct {
	cache.Cache[K, V]
	swrManyCalls int
}

type countingConnectionManager struct {
	analytics.ConnectionManager
	calls int
}

func (m *countingConnectionManager) GetConnection(ctx context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	m.calls++
	return m.ConnectionManager.GetConnection(ctx, workspaceID)
}

func (c *countingCache[K, V]) SWRMany(
	ctx context.Context,
	keys []K,
	refreshFromOrigin func(context.Context, []K) (map[K]V, error),
	op func(error) cache.Op,
) (map[K]V, map[K]cache.CacheHit, error) {
	c.swrManyCalls++
	return c.Cache.SWRMany(ctx, keys, refreshFromOrigin, op)
}

// Test400_KeySpaceLimitPrecedesAuthorizationLookups guarantees the eleventh unique ID is rejected before cache or database fan-out.
func Test400_KeySpaceLimitPrecedesAuthorizationLookups(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.api_test.read_analytics")

	apiLookups := &countingCache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{Cache: h.Caches.ApiToKeyAuthRow}
	keySpaceLookups := &countingCache[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]{Cache: h.Caches.KeyAuthToApiRow}
	connectionLookups := &countingConnectionManager{ConnectionManager: h.AnalyticsConnectionManager}
	routeCaches := h.Caches
	routeCaches.ApiToKeyAuthRow = apiLookups
	routeCaches.KeyAuthToApiRow = keySpaceLookups
	route := &Handler{DB: h.DB, AnalyticsConnectionManager: connectionLookups, Caches: routeCaches}
	h.Register(route)

	ids := make([]string, 11)
	for i := range ids {
		ids[i] = fmt.Sprintf("'ks_%d' = v.key_space_id", i)
	}
	res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, http.Header{
		"Authorization": []string{"Bearer " + rootKey},
		"Content-Type":  []string{"application/json"},
	}, Request{Query: "SELECT COUNT(*) FROM key_verifications_v1 AS v WHERE " + strings.Join(ids, " OR ")})

	require.Equal(t, http.StatusBadRequest, res.Status)
	require.Zero(t, connectionLookups.calls)
	require.Zero(t, apiLookups.swrManyCalls)
	require.Zero(t, keySpaceLookups.swrManyCalls)
}

// Test400_QueryWorkBoundsPrecedeAuthorizationLookups guarantees invalid query work is bounded before cache or database lookups.
func Test400_QueryWorkBoundsPrecedeAuthorizationLookups(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "query length",
			query: "SELECT 1 " + strings.Repeat(" ", 16*1024),
		},
		{
			name:  "AST complexity",
			query: "SELECT " + strings.Repeat("1,", 600) + "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
			workspace := h.CreateWorkspace()
			h.SetupAnalytics(workspace.ID)
			rootKey := h.CreateRootKey(workspace.ID, "api.api_test.read_analytics")

			apiLookups := &countingCache[cache.ScopedKey, db.FindKeyAuthsByIdsRow]{Cache: h.Caches.ApiToKeyAuthRow}
			keySpaceLookups := &countingCache[cache.ScopedKey, db.FindKeyAuthsByKeyAuthIdsRow]{Cache: h.Caches.KeyAuthToApiRow}
			connectionLookups := &countingConnectionManager{ConnectionManager: h.AnalyticsConnectionManager}
			routeCaches := h.Caches
			routeCaches.ApiToKeyAuthRow = apiLookups
			routeCaches.KeyAuthToApiRow = keySpaceLookups
			route := &Handler{DB: h.DB, AnalyticsConnectionManager: connectionLookups, Caches: routeCaches}
			h.Register(route)

			res := testutil.CallRoute[Request, openapi.BadRequestErrorResponse](h, route, http.Header{
				"Authorization": []string{"Bearer " + rootKey},
				"Content-Type":  []string{"application/json"},
			}, Request{Query: tt.query})

			require.Equal(t, http.StatusBadRequest, res.Status)
			require.Zero(t, connectionLookups.calls)
			require.Zero(t, apiLookups.swrManyCalls)
			require.Zero(t, keySpaceLookups.swrManyCalls)
		})
	}
}

func Test400_EmptyQuery(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID)
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})

	workspace := h.CreateWorkspace()
	h.SetupAnalytics(workspace.ID, testutil.WithRetentionDays(90)) // 90-day retention
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_analytics")

	route := &Handler{
		DB:                         h.DB,
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
