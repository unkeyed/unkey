package internalCacheInvalidate_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/internal_cache_invalidate"
)

func TestMissingAuthorizationHeader(t *testing.T) {
	h := testutil.NewHarness(t)
	route := setupRoute(h)

	headers := http.Header{
		"Content-Type": {"application/json"},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		CacheName: "verification_key_by_hash",
		Keys:      []string{"some_key"},
	})

	require.Equal(t, http.StatusBadRequest, res.Status)
}

func TestEmptyCacheName(t *testing.T) {
	h := testutil.NewHarness(t)
	route := setupRoute(h)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, validHeaders(), handler.Request{
		CacheName: "",
		Keys:      []string{"some_key"},
	})

	require.Equal(t, http.StatusBadRequest, res.Status)
}
