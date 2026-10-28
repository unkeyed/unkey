package v2RatelimitLimit_test

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
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/db"
	ratelimitdb "github.com/unkeyed/unkey/pkg/ratelimit/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
)

// The regional limiter uses fixed windows, so an exact reset time and a unit
// that comes back after `Duration / Limit` show that the global limiter
// decided.
func TestGlobalNamespace(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		RatelimitEvents: h.RatelimitEvents,
		DirectAuditLogs: h.DirectAuditLogs,
		Ratelimit:       h.Ratelimit,
		GlobalRatelimit: h.GlobalRatelimit,
		DB:              h.DB,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	createGlobalNamespace := func(t *testing.T, mode db.RatelimitNamespacesGlobalRatelimitMode) (headers http.Header, namespaceID, namespaceName string) {
		t.Helper()
		namespaceID, namespaceName = createNamespace(t, h)
		err := db.Query.UpdateRatelimitNamespaceGlobalRatelimitMode(ctx, h.DB.RW(), db.UpdateRatelimitNamespaceGlobalRatelimitModeParams{
			GlobalRatelimitMode: mode,
			UpdatedAtM:          sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			WorkspaceID:         h.Resources().UserWorkspace.ID,
			ID:                  namespaceID,
		})
		require.NoError(t, err)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))
		headers = http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		return headers, namespaceID, namespaceName
	}

	t.Run("allows up to the limit, then denies until a unit is free", func(t *testing.T) {
		headers, namespaceID, namespaceName := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: uid.New(uid.TestPrefix),
			Limit:      2,
			Duration:   60_000,
		}
		start := h.Clock.Now()

		for _, wantRemaining := range []int64{1, 0} {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.True(t, res.Body.Data.Success)
			require.Equal(t, int64(2), res.Body.Data.Limit)
			require.Equal(t, wantRemaining, res.Body.Data.Remaining)
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.False(t, res.Body.Data.Success)
		require.Equal(t, int64(0), res.Body.Data.Remaining)
		require.Equal(t, start.Add(time.Minute).UnixMilli(), res.Body.Data.Reset)

		// Units that MySQL has not acknowledged by the end of their bucket's
		// writes count as lost, so the flush must land before the clock
		// jumps.
		h.Clock.Tick(time.Second)
		fingerprint := namespace.GlobalFingerprint(h.Resources().UserWorkspace.ID, namespaceID, req.Identifier, time.Minute)
		require.Eventually(t, func() bool {
			buckets, err := ratelimitdb.Query.ListBuckets(ctx, h.DB.RO(), ratelimitdb.ListBucketsParams{PodID: "", Fingerprint: fingerprint[:], Limit: 10})
			return err == nil && len(buckets) == 1 && buckets[0].PeerConsumedNs == time.Minute.Nanoseconds()
		}, 5*time.Second, 10*time.Millisecond)

		h.Clock.Tick(29 * time.Second)
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(0), res.Body.Data.Remaining)
	})

	t.Run("applies the override limit and duration", func(t *testing.T) {
		headers, namespaceID, namespaceName := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeOn)
		identifier := uid.New(uid.TestPrefix)
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			Limit:       4,
			Duration:    120_000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
			Limit:      1,
			Duration:   60_000,
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, overrideID, res.Body.Data.OverrideId)
		require.Equal(t, int64(4), res.Body.Data.Limit)
		require.Equal(t, int64(3), res.Body.Data.Remaining)
		require.Equal(t, h.Clock.Now().Add(30*time.Second).UnixMilli(), res.Body.Data.Reset)
	})

	t.Run("shadow mode responds with the regional limiter and counts the global decision", func(t *testing.T) {
		headers, _, namespaceName := createGlobalNamespace(t, db.RatelimitNamespacesGlobalRatelimitModeShadow)
		workspaceID := h.Resources().UserWorkspace.ID
		globalCount := func(global string) float64 {
			count := 0.0
			for _, regional := range []string{"passed", "denied"} {
				var metric dto.Metric
				require.NoError(t, metrics.RatelimitShadowDecisions.WithLabelValues(workspaceID, "limit", regional, global).Write(&metric))
				count += metric.GetCounter().GetValue()
			}
			return count
		}
		passedBefore := globalCount("passed")
		deniedBefore := globalCount("denied")
		start := h.Clock.Set(h.Clock.Now().Truncate(time.Minute).Add(time.Minute + 7*time.Millisecond))
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: uid.New(uid.TestPrefix),
			Limit:      2,
			Duration:   60_000,
		}

		for _, wantSuccess := range []bool{true, true, false} {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, wantSuccess, res.Body.Data.Success)
			require.Equal(t, start.Truncate(time.Minute).Add(time.Minute).UnixMilli(), res.Body.Data.Reset, "the regional fixed window must decide")
		}

		// The shadow decisions run concurrently, so any one of them can be the
		// denied one. One is only denied if the others spent units.
		require.Eventually(t, func() bool {
			return globalCount("passed") == passedBefore+2 && globalCount("denied") == deniedBefore+1
		}, 5*time.Second, 10*time.Millisecond)
	})
}
