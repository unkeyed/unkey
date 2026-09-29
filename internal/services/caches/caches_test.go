package caches_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

// TestRemovingVerificationKeyInvalidatesRootKey guarantees mutations to legacy
// keys evict the same hash from both typed authentication caches.
func TestRemovingVerificationKeyInvalidatesRootKey(t *testing.T) {
	c, err := caches.New(caches.Config{Clock: clock.NewTestClock()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Close()) })

	const hash = "shared_hash"
	c.VerificationKeyByHash.Set(t.Context(), hash, keysdb.CachedKeyData{})
	c.RootKeyByHash.Set(t.Context(), hash, keysdb.CachedRootKeyData{})

	c.VerificationKeyByHash.Remove(t.Context(), hash)

	_, verificationHit := c.VerificationKeyByHash.Get(t.Context(), hash)
	_, rootHit := c.RootKeyByHash.Get(t.Context(), hash)
	require.Equal(t, cache.Miss, verificationHit)
	require.Equal(t, cache.Miss, rootHit)
}
