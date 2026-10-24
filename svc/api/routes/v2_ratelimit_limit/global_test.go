package v2RatelimitLimit_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
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

	createGlobalNamespace := func(t *testing.T) (headers http.Header, namespaceID, namespaceName string) {
		t.Helper()
		namespaceID, namespaceName = createNamespace(t, h)
		err := db.Query.UpdateRatelimitNamespaceUseGlobalRatelimit(ctx, h.DB.RW(), db.UpdateRatelimitNamespaceUseGlobalRatelimitParams{
			UseGlobalRatelimit: true,
			UpdatedAtM:         sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			WorkspaceID:        h.Resources().UserWorkspace.ID,
			ID:                 namespaceID,
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
		headers, _, namespaceName := createGlobalNamespace(t)
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

		h.Clock.Tick(30 * time.Second)
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(0), res.Body.Data.Remaining)
	})

	t.Run("applies the override limit and duration", func(t *testing.T) {
		headers, namespaceID, namespaceName := createGlobalNamespace(t)
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
}
