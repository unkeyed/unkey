package internalCacheInvalidate_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/internal_cache_invalidate"
)

func TestInvalidBearerToken(t *testing.T) {
	h := testutil.NewHarness(t)
	route := setupRoute(h)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer wrong-token"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		CacheName: "verification_key_by_hash",
		Keys:      []string{"some_key"},
	})

	require.Equal(t, http.StatusUnauthorized, res.Status)
}
