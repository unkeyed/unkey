package internalCacheInvalidate_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/internal_cache_invalidate"
)

func TestUnknownCacheName(t *testing.T) {
	h := testutil.NewHarness(t)
	route := setupRoute(h)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, validHeaders(), handler.Request{
		CacheName: "nonexistent_cache",
		Keys:      []string{"some_key"},
	})

	require.Equal(t, http.StatusInternalServerError, res.Status)
}
