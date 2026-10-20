package v2_ratelimit_multi_limit_test

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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

func TestMultiLimitWithGCRAIsAllOrNothing(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{})
	route := &handler.Handler{
		RatelimitEvents: h.RatelimitEvents,
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
	roomy := handler.Request{{Namespace: namespaceName, Identifier: "roomy", Limit: 100, Duration: 600_000, Cost: new(int64(4))}}
	tight := handler.Request{{Namespace: namespaceName, Identifier: "tight", Limit: 1, Duration: 60_000}}

	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, tight)
	require.Equal(t, 200, first.Status, "expected 200, received: %s", first.RawBody)
	require.True(t, first.Body.Data.Passed)

	both := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, append(append(handler.Request{}, roomy...), tight...))
	require.Equal(t, 200, both.Status)
	require.False(t, both.Body.Data.Passed)
	require.True(t, both.Body.Data.Limits[0].Passed)
	require.Equal(t, int64(10), both.Body.Data.Limits[0].Remaining, "nothing was spent, so its own cost is reported back")
	require.False(t, both.Body.Data.Limits[1].Passed)

	alone := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, roomy)
	require.True(t, alone.Body.Data.Passed)
	require.Equal(t, int64(6), alone.Body.Data.Limits[0].Remaining, "the failed call spent nothing")
}
