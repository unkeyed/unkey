package internalCacheInvalidate_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/internal_cache_invalidate"
)

func TestSuccessfulInvalidation(t *testing.T) {
	h := testutil.NewHarness(t)
	route := setupRoute(h)
	ctx := context.Background()

	// Populate the verification_key_by_hash cache with a test entry.
	testHash := "test_hash_for_invalidation"
	h.Caches.VerificationKeyByHash.Set(ctx, testHash, db.CachedKeyData{})

	// Verify the key is in the cache.
	_, hit := h.Caches.VerificationKeyByHash.Get(ctx, testHash)
	require.Equal(t, cache.Hit, hit, "key should be in cache before invalidation")

	// Call the invalidation endpoint.
	res := testutil.CallRoute[handler.Request, handler.Response](h, route, validHeaders(), handler.Request{
		CacheName: "verification_key_by_hash",
		Keys:      []string{testHash},
	})

	require.Equal(t, http.StatusOK, res.Status)

	// Verify the key was evicted from the cache.
	_, hit = h.Caches.VerificationKeyByHash.Get(ctx, testHash)
	require.Equal(t, cache.Miss, hit, "key should be evicted after invalidation")
}
