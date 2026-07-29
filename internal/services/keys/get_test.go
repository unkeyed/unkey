package keys

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

type synchronizedMissCache struct {
	cache.Cache[string, keysdb.CachedKeyData]
	callers    int32
	arrived    atomic.Int32
	allArrived chan struct{}
}

func (c *synchronizedMissCache) SWR(
	ctx context.Context,
	_ string,
	refreshFromOrigin func(context.Context) (keysdb.CachedKeyData, error),
	_ func(error) cache.Op,
) (keysdb.CachedKeyData, cache.CacheHit, error) {
	if c.arrived.Add(1) == c.callers {
		close(c.allArrived)
	}
	<-c.allArrived

	value, err := refreshFromOrigin(ctx)
	return value, cache.Hit, err
}

type blockingKeyQuerier struct {
	keysdb.Querier
	calls   atomic.Int32
	started chan struct{}
	release chan struct{}
}

func (q *blockingKeyQuerier) FindKeyForVerification(
	context.Context,
	keysdb.DBTX,
	string,
) (keysdb.FindKeyForVerificationRow, error) {
	if q.calls.Add(1) == 1 {
		close(q.started)
	}
	<-q.release

	return keysdb.FindKeyForVerificationRow{
		ID:               "key_123",
		KeyAuthID:        "keyauth_123",
		WorkspaceID:      "ws_123",
		ApiWorkspaceID:   "ws_123",
		ApiID:            "api_123",
		Roles:            []byte("[]"),
		Permissions:      []byte("[]"),
		Ratelimits:       []byte("[]"),
		Enabled:          true,
		WorkspaceEnabled: true,
	}, nil
}

func TestGetRootKey_ErrorHandling_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create a service with nil dependencies, similar to create_test.go pattern
	s := &service{}

	ctx := context.Background()

	// This test validates that GetRootKey properly handles errors from Get()
	// Before the fix, GetRootKey would panic when trying to access key.Key.ForWorkspaceID.Valid
	// after Get() returned an error with nil key. Now it should return the error safely.
	key, err := s.GetRootKey(ctx, nil)

	require.Error(t, err)
	require.Nil(t, key)

	// Verify specific error code for missing auth when session is nil
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Auth.Authentication.Missing.URN(), code)
}

func TestGetRootKey_WithEmptyRawKey_ReturnsError(t *testing.T) {
	t.Parallel()

	// Create a service with nil dependencies, following create_test.go pattern
	s := &service{}

	ctx := context.Background()

	// Call Get with empty raw key to test the assert.NotEmpty validation
	key, err := s.Get(ctx, nil, "")

	// Verify that we get an error for empty key
	require.Error(t, err)
	require.Nil(t, key)
	require.Contains(t, err.Error(), "sha256Hash is empty")
}

func TestGet_WithEmptyRawKey_ReturnsError(t *testing.T) {
	t.Parallel()

	// Test the assert.NotEmpty validation path directly in Get function
	s := &service{}
	ctx := context.Background()

	key, err := s.Get(ctx, nil, "")

	require.Error(t, err)
	require.Nil(t, key)
	require.Contains(t, err.Error(), "sha256Hash is empty")
}

func TestGet_EmptyString_Variants(t *testing.T) {
	t.Parallel()

	// Test various empty string cases to improve assert.NotEmpty coverage
	s := &service{}
	ctx := context.Background()

	// Only test cases that will hit the validation path, not the cache/db path
	emptyVariants := []string{
		"", // Classic empty string
	}

	for _, empty := range emptyVariants {
		key, err := s.Get(ctx, nil, empty)

		require.Error(t, err)
		require.Nil(t, key)
		require.Contains(t, err.Error(), "sha256Hash is empty")
	}
}

func TestGet_CoalescesConcurrentCacheMisses(t *testing.T) {
	const callers = 100

	originalQuerier := keysdb.Query
	release := make(chan struct{})
	var releaseOnce sync.Once
	querier := &blockingKeyQuerier{
		Querier: originalQuerier,
		started: make(chan struct{}),
		release: release,
	}
	keysdb.Query = querier
	t.Cleanup(func() {
		keysdb.Query = originalQuerier
	})
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
	})

	keyCache := &synchronizedMissCache{
		Cache:      cache.NewNoopCache[string, keysdb.CachedKeyData](),
		callers:    callers,
		allArrived: make(chan struct{}),
	}
	s := &service{
		db:       &keysdb.Database{},
		keyCache: keyCache,
	}

	type result struct {
		verifier *KeyVerifier
		err      error
	}
	results := make(chan result, callers)
	for range callers {
		go func() {
			verifier, err := s.Get(context.Background(), nil, "hash")
			results <- result{verifier: verifier, err: err}
		}()
	}

	select {
	case <-querier.started:
	case <-time.After(time.Second):
		t.Fatal("database load did not start")
	}

	require.Never(t, func() bool {
		return querier.calls.Load() > 1
	}, 100*time.Millisecond, time.Millisecond, "concurrent misses must share one database load")
	releaseOnce.Do(func() { close(release) })

	for range callers {
		res := <-results
		require.NoError(t, res.err)
		require.NotNil(t, res.verifier)
		require.Equal(t, StatusValid, res.verifier.Status)
	}
	require.Equal(t, int32(1), querier.calls.Load())
}
