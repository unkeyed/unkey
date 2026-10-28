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
	setoverride "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"
)

func TestWorkspaceRateLimit_DefaultLimits_Unlimited(t *testing.T) {
	h := testutil.NewHarness(t)

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	route := &setoverride.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := setoverride.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
		Limit:      10,
		Duration:   1000,
	}

	res := testutil.CallRoute[setoverride.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status)
}

func TestWorkspaceRateLimit_NullFields_Unlimited(t *testing.T) {
	h := testutil.NewHarness(t)

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	upsertWorkspaceLimits(t, h, ws.ID, sql.NullInt32{})

	route := &setoverride.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := setoverride.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
		Limit:      10,
		Duration:   1000,
	}

	res := testutil.CallRoute[setoverride.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status)
}

func TestWorkspaceRateLimit_ZeroLimit_AllowsReads(t *testing.T) {
	h := testutil.NewHarness(t)

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

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

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status)
	require.Empty(t, res.Headers.Get("RateLimit-Limit"))
}

func TestWorkspaceRateLimit_ReadsDoNotConsumeWriteAllowance(t *testing.T) {
	h := testutil.NewHarness(t)

	ws := h.CreateWorkspace()
	rootKey := h.CreateRootKey(ws.ID)

	upsertWorkspaceLimits(t, h, ws.ID, sql.NullInt32{Valid: true, Int32: 2})

	route := &handler.Handler{
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(route)
	writeRoute := &setoverride.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}
	h.Register(writeRoute)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		Namespace:  "nonexistent",
		Identifier: "test",
	}

	for range 3 {
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.Empty(t, res.Headers.Get("RateLimit-Limit"))
	}

	writeRequest := setoverride.Request{
		Namespace:  req.Namespace,
		Identifier: req.Identifier,
		Limit:      10,
		Duration:   1000,
	}
	for range 2 {
		res := testutil.CallRoute[setoverride.Request, openapi.NotFoundErrorResponse](h, writeRoute, headers, writeRequest)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "2", res.Headers.Get("RateLimit-Limit"))
	}

	blocked := testutil.CallRoute[setoverride.Request, openapi.TooManyRequestsErrorResponse](h, writeRoute, headers, writeRequest)
	require.Equal(t, http.StatusTooManyRequests, blocked.Status)
	require.Equal(t, "https://unkey.com/docs/errors/user/too_many_requests/workspace_rate_limited", blocked.Body.Error.Type)

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status)
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
		AutoscalingReplicasMax:                0,
	})
	require.NoError(t, err)
}
