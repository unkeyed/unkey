package caches

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

// Cache invalidation is process-local: deleting or disabling a key clears only
// the cache of the node that handled the mutation, so every other node keeps
// serving the cached key until its entry passes the stale window and the next
// lookup revalidates against MySQL. The stale window is therefore the upper
// bound on how long a key revoked on one node keeps verifying on another, and
// it must stay inside the window Unkey documents for global revocation.
func TestVerificationKeyByHashBoundsRevocationLag(t *testing.T) {
	t.Parallel()

	// Unkey documents revoked keys as stopping within 30 seconds globally.
	documentedRevocationWindow := 30 * time.Second

	clk := clock.NewTestClock(time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC))
	caches, err := New(Config{Clock: clk})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, caches.Close()) })

	ctx := context.Background()
	hash := "sha256-of-the-revoked-key"

	revoked := false
	refreshFromOrigin := func(context.Context) (keysdb.CachedKeyData, error) {
		if revoked {
			return keysdb.CachedKeyData{}, sql.ErrNoRows
		}
		return keysdb.CachedKeyData{ //nolint:exhaustruct // only the row identity matters here
			FindKeyForVerificationRow: keysdb.FindKeyForVerificationRow{ //nolint:exhaustruct // only the row identity matters here
				ID:          "key_123",
				WorkspaceID: "ws_123",
			},
		}, nil
	}

	value, hit, err := caches.VerificationKeyByHash.SWR(ctx, hash, refreshFromOrigin, DefaultFindFirstOp)
	require.NoError(t, err)
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "key_123", value.ID)

	revoked = true

	clk.Tick(documentedRevocationWindow + time.Second)

	_, hit, err = caches.VerificationKeyByHash.SWR(ctx, hash, refreshFromOrigin, DefaultFindFirstOp)
	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NotEqual(t, cache.Hit, hit)
}
