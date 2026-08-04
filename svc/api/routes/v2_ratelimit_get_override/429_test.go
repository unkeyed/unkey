package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_get_override"
)

func TestWorkspaceRateLimit_DefaultLimits_Unlimited(t *testing.T) {
	h := testutil.NewHarness(t)

	// Fresh workspaces have no workspace API throttle configured by default.
	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	// Should not be 429. The handler proceeds past rate limiting and hits
	// namespace-not-found.
	require.NotEqual(t, http.StatusTooManyRequests, res.Status)
}

func TestWorkspaceRateLimit_NullFields_Unlimited(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	// Limits exist but the API request limit is NULL = unlimited.
	err := db.Query.UpsertQuota(ctx, h.DB.RW(), db.UpsertQuotaParams{
		WorkspaceID:            ws.ID,
		RequestsPerMonth:       1_000_000,
		AuditLogsRetentionDays: 30,
		LogsRetentionDays:      30,
		Team:                   false,
		RatelimitApiLimit:      sql.NullInt32{}, //nolint:exhaustruct
		RatelimitApiDuration:   sql.NullInt32{}, //nolint:exhaustruct
	})
	require.NoError(t, err)
	upsertWorkspaceLimits(t, h, ws.ID, sql.NullInt32{})

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.NotEqual(t, http.StatusTooManyRequests, res.Status)
}

func TestWorkspaceRateLimit_ZeroLimit_Returns429(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	// limit=0 means explicitly blocked, zero requests allowed
	err := db.Query.UpsertQuota(ctx, h.DB.RW(), db.UpsertQuotaParams{
		WorkspaceID:            ws.ID,
		RequestsPerMonth:       1_000_000,
		AuditLogsRetentionDays: 30,
		LogsRetentionDays:      30,
		Team:                   false,
		RatelimitApiLimit:      sql.NullInt32{Valid: true, Int32: 0},
		RatelimitApiDuration:   sql.NullInt32{Valid: true, Int32: 60000},
	})
	require.NoError(t, err)
	upsertWorkspaceLimits(t, h, ws.ID, sql.NullInt32{Valid: true, Int32: 0})

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
	}

	res := testutil.CallRoute[handler.Request, openapi.TooManyRequestsErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusTooManyRequests, res.Status)
	require.Equal(t, http.StatusTooManyRequests, res.Body.Error.Status)
}

func TestWorkspaceRateLimit_EnforcesLimit(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := context.Background()

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	// Allow exactly 2 requests per 60s window
	err := db.Query.UpsertQuota(ctx, h.DB.RW(), db.UpsertQuotaParams{
		WorkspaceID:            ws.ID,
		RequestsPerMonth:       1_000_000,
		AuditLogsRetentionDays: 30,
		LogsRetentionDays:      30,
		Team:                   false,
		RatelimitApiLimit:      sql.NullInt32{Valid: true, Int32: 2},
		RatelimitApiDuration:   sql.NullInt32{Valid: true, Int32: 60000},
	})
	require.NoError(t, err)
	upsertWorkspaceLimits(t, h, ws.ID, sql.NullInt32{Valid: true, Int32: 2})

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
	}

	// First two requests should pass rate limiting
	res1 := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.NotEqual(t, http.StatusTooManyRequests, res1.Status)

	res2 := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.NotEqual(t, http.StatusTooManyRequests, res2.Status)

	// Third request should be rate limited
	res3 := testutil.CallRoute[handler.Request, openapi.TooManyRequestsErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusTooManyRequests, res3.Status)
}

func upsertWorkspaceLimits(
	t *testing.T,
	h *testutil.Harness,
	workspaceID string,
	apiRequestsCountMaxPerMinute sql.NullInt32,
) {
	t.Helper()

	err := db.Query.UpsertLimit(context.Background(), h.DB.RW(), db.UpsertLimitParams{
		WorkspaceID:                           workspaceID,
		ApiBillableOperationsCountMaxPerMonth: 1_000_000,
		ApiRequestsCountMaxPerMinute:          apiRequestsCountMaxPerMinute,
		LogsRetentionDaysMax:                  30,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           10,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          20_480,
		MemoryMibMaxPerInstance:               4_096,
		StorageMibMax:                         51_200,
		StorageMibMaxPerInstance:              10_240,
		BuildsConcurrentMax:                   1,
		CustomDomainsMax:                      0,
		AutoscalingReplicasMax:                4,
	})
	require.NoError(t, err)
}
