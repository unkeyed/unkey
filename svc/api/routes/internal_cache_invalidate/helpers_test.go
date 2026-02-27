package internalCacheInvalidate_test

import (
	"net/http"

	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/internal_cache_invalidate"
)

const testToken = "test-cache-invalidation-token"

func setupRoute(h *testutil.Harness) *handler.Handler {
	route := &handler.Handler{
		Caches: h.Caches,
		Token:  testToken,
	}
	h.Register(route)
	return route
}

func validHeaders() http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + testToken},
	}
}
