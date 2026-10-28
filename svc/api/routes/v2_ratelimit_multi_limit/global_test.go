package v2_ratelimit_multi_limit_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

// The regional limiter resets at the end of a fixed window, a multiple of the
// duration. The global limiter resets at an exact time, `Duration / Limit`
// per spent unit after now. Each test starts 7 ms after a minute, so the two
// never agree.
func TestGlobalNamespaces(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		RatelimitEvents: h.RatelimitEvents,
		Ratelimit:       h.Ratelimit,
		GlobalRatelimit: h.GlobalRatelimit,
		DB:              h.DB,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.limit")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	createGlobalNamespace := func(t *testing.T, mode db.RatelimitNamespacesGlobalRatelimitMode) (namespaceID, namespaceName string) {
		t.Helper()
		namespaceID, namespaceName = createNamespace(t, h)
		err := db.Query.UpdateRatelimitNamespaceGlobalRatelimitMode(ctx, h.DB.RW(), db.UpdateRatelimitNamespaceGlobalRatelimitModeParams{
			GlobalRatelimitMode: mode,
			UpdatedAtM:          sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			WorkspaceID:         h.Resources().UserWorkspace.ID,
			ID:                  namespaceID,
		})
		require.NoError(t, err)
		return namespaceID, namespaceName
	}

	startAfterMinute := func() time.Time {
		return h.Clock.Set(h.Clock.Now().Truncate(time.Minute).Add(time.Minute + 7*time.Millisecond))
	}

	t.Run("spends nothing when one check is denied", func(t *testing.T) {
		_, first := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		_, second := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		identifier := uid.New(uid.TestPrefix)
		start := startAfterMinute()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			{Namespace: first, Identifier: identifier, Limit: 1, Duration: 60_000},
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Passed)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			{Namespace: first, Identifier: identifier, Limit: 1, Duration: 60_000},
			{Namespace: second, Identifier: identifier, Limit: 2, Duration: 60_000},
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.False(t, res.Body.Data.Passed)
		require.False(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, start.Add(time.Minute).UnixMilli(), res.Body.Data.Limits[0].Reset)
		require.True(t, res.Body.Data.Limits[1].Passed)

		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			{Namespace: second, Identifier: identifier, Limit: 2, Duration: 60_000},
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Passed)
		require.Equal(t, int64(1), res.Body.Data.Limits[0].Remaining, "the denied call must not have spent a unit")
		require.Equal(t, start.Add(30*time.Second).UnixMilli(), res.Body.Data.Limits[0].Reset)
	})

	t.Run("a call with a regional namespace stays regional", func(t *testing.T) {
		_, global := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		_, regional := createNamespace(t, h)
		identifier := uid.New(uid.TestPrefix)
		start := startAfterMinute()

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			{Namespace: global, Identifier: identifier, Limit: 2, Duration: 60_000},
			{Namespace: regional, Identifier: identifier, Limit: 2, Duration: 60_000},
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Passed)
		windowEnd := start.Truncate(time.Minute).Add(time.Minute).UnixMilli()
		require.Equal(t, windowEnd, res.Body.Data.Limits[0].Reset)
		require.Equal(t, windowEnd, res.Body.Data.Limits[1].Reset)
	})

	t.Run("rejects two checks that an override makes equal", func(t *testing.T) {
		namespaceID, namespaceName := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		identifier := uid.New(uid.TestPrefix)
		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          uid.New(uid.RatelimitOverridePrefix),
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			Limit:       10,
			Duration:    120_000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
			{Namespace: namespaceName, Identifier: identifier, Limit: 5, Duration: 60_000},
			{Namespace: namespaceName, Identifier: identifier, Limit: 5, Duration: 30_000},
		})
		require.Equal(t, 400, res.Status, "expected 400, received: %s", res.RawBody)
	})

	t.Run("shadow mode responds with the regional limiter and counts the global decision", func(t *testing.T) {
		_, shadow := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeShadow)
		_, global := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		workspaceID := h.Resources().UserWorkspace.ID
		globalCount := func(global string) float64 {
			count := 0.0
			for _, regional := range []string{"passed", "denied"} {
				var metric dto.Metric
				require.NoError(t, metrics.RatelimitShadowDecisions.WithLabelValues(workspaceID, "multi_limit", regional, global).Write(&metric))
				count += metric.GetCounter().GetValue()
			}
			return count
		}
		passedBefore := globalCount("passed")
		deniedBefore := globalCount("denied")
		identifier := uid.New(uid.TestPrefix)
		start := startAfterMinute()
		req := handler.Request{
			{Namespace: shadow, Identifier: identifier, Limit: 1, Duration: 60_000},
			{Namespace: global, Identifier: identifier, Limit: 2, Duration: 60_000},
		}

		for _, wantPassed := range []bool{true, false} {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, wantPassed, res.Body.Data.Passed)
			windowEnd := start.Truncate(time.Minute).Add(time.Minute).UnixMilli()
			require.Equal(t, windowEnd, res.Body.Data.Limits[0].Reset, "the regional fixed window must decide")
			require.Equal(t, windowEnd, res.Body.Data.Limits[1].Reset, "the regional fixed window must decide")
		}

		// The shadow decisions run concurrently, so either one can be the denied
		// one. One is only denied if the other spent units.
		require.Eventually(t, func() bool {
			return globalCount("passed") == passedBefore+1 && globalCount("denied") == deniedBefore+1
		}, 5*time.Second, 10*time.Millisecond)
	})
}
