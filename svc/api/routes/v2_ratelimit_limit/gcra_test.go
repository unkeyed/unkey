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
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
)

// TestLimitWithGCRA checks what tells GCRA apart from the sliding window: the
// reset is when the spent time is paid off, not a window boundary, and a
// unit's worth of time frees one unit again within the same window.
func TestLimitWithGCRA(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{})
	route := &handler.Handler{
		RatelimitEvents: h.RatelimitEvents,
		DirectAuditLogs: h.DirectAuditLogs,
		Ratelimit:       h.Ratelimit,
		DB:              h.DB,
		NamespaceCache:  h.Caches.RatelimitNamespace,
		Auditlogs:       h.Auditlogs,
	}
	h.Register(route)

	namespaceID, namespaceName := createNamespace(t, h)
	require.NoError(t, db.Query.UpdateRatelimitNamespaceUseGcra(context.Background(), h.DB.RW(), db.UpdateRatelimitNamespaceUseGcraParams{
		UseGcra: true,
		Now:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		ID:      namespaceID,
	}))
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	req := handler.Request{Namespace: namespaceName, Identifier: "user_123", Limit: 10, Duration: 10_000}

	// One second into a ten-second window, so the next second stays in it.
	start := time.UnixMilli((h.Clock.Now().UnixMilli()/10_000+1)*10_000 + 1_000)
	h.Clock.Set(start)

	for i := range 10 {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(9-i), res.Body.Data.Remaining)
		require.Equal(t, start.Add(time.Duration(i+1)*time.Second).UnixMilli(), res.Body.Data.Reset)
	}

	denied := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, denied.Status)
	require.False(t, denied.Body.Data.Success)
	require.Equal(t, int64(0), denied.Body.Data.Remaining)
	require.Equal(t, start.Add(10*time.Second).UnixMilli(), denied.Body.Data.Reset)

	h.Clock.Tick(time.Second)
	refilled := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.True(t, refilled.Body.Data.Success, "one second frees one unit")
	again := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.False(t, again.Body.Data.Success)
}
