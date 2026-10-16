package cache_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
)

func TestSWR_CacheHit(t *testing.T) {
	ctx := context.Background()
	mockClock := clock.NewTestClock()

	c, err := cache.New(cache.Config[string, string]{
		Fresh:    1 * time.Minute,
		Stale:    5 * time.Minute,
		MaxSize:  100,
		Resource: "test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	t.Run("miss on first call", func(t *testing.T) {
		value, hit, err := c.SWR(ctx, "key1", func(ctx context.Context) (string, error) {
			return "value1", nil
		}, func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value1", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("hit on subsequent call within fresh time", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key2", func(ctx context.Context) (string, error) {
			return "value2", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Second call should hit cache
		value, hit, err := c.SWR(ctx, "key2", func(ctx context.Context) (string, error) {
			t.Fatal("should not call refresh function")
			return "", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value2", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("null cache hit", func(t *testing.T) {
		// First call returns not found error
		_, _, err := c.SWR(ctx, "key3", func(ctx context.Context) (string, error) {
			return "", sql.ErrNoRows
		}, func(err error) cache.Op {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}
			return cache.Noop
		})
		require.Error(t, err)
		require.True(t, db.IsNotFound(err))

		// Second call should return null hit
		value, hit, err := c.SWR(ctx, "key3", func(ctx context.Context) (string, error) {
			t.Fatal("should not call refresh function")
			return "", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "", value)
		require.Equal(t, cache.Null, hit)
	})

	t.Run("stale hit returns cached value", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key4", func(ctx context.Context) (string, error) {
			return "value4", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Move time forward past fresh but within stale
		mockClock.Tick(2 * time.Minute)

		// Should return cached value with hit status
		value, hit, err := c.SWR(ctx, "key4", func(ctx context.Context) (string, error) {
			// This will be called in background
			return "updated_value4", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "value4", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("miss after stale time", func(t *testing.T) {
		// First call to populate cache
		_, _, err := c.SWR(ctx, "key5", func(ctx context.Context) (string, error) {
			return "value5", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})
		require.NoError(t, err)

		// Move time forward past stale
		mockClock.Tick(6 * time.Minute)

		// Should call refresh and return new value
		value, hit, err := c.SWR(ctx, "key5", func(ctx context.Context) (string, error) {
			return "new_value5", nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Equal(t, "new_value5", value)
		require.Equal(t, cache.Hit, hit)
	})

	t.Run("error returns miss", func(t *testing.T) {
		expectedErr := errors.New("refresh error")
		value, hit, err := c.SWR(ctx, "key6", func(ctx context.Context) (string, error) {
			return "", expectedErr
		}, func(err error) cache.Op {
			return cache.Noop
		})

		require.Error(t, err)
		require.Equal(t, expectedErr, err)
		require.Equal(t, "", value)
		require.Equal(t, cache.Miss, hit)
	})
}

func newFallbackCache(t *testing.T) (cache.Cache[string, string], *clock.TestClock) {
	t.Helper()
	clk := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  100,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c, clk
}

// TestSWRWithFallbackServesFirstLiveCandidate guarantees that candidates are
// checked in order and that missing or expired candidates are skipped, so a
// wildcard entry still serves a host whose exact entry is gone.
func TestSWRWithFallbackServesFirstLiveCandidate(t *testing.T) {
	c, clk := newFallbackCache(t)
	c.Set(t.Context(), "expired", "expired-value")
	clk.Tick(2 * time.Hour)
	c.Set(t.Context(), "live", "live-value")
	c.Set(t.Context(), "later", "later-value")

	value, hit, err := c.SWRWithFallback(t.Context(), []string{"missing", "expired", "live", "later"}, func(context.Context) (string, string, error) {
		t.Error("a live candidate must be served without reading the origin")
		return "", "", nil
	}, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "live-value", value)
}

// TestSWRWithFallbackCachesUnderCanonicalKey guarantees that a miss on every
// candidate returns the origin value and stores it under the key the origin
// names, as op decides, and that Noop reports a miss rather than a cached null.
func TestSWRWithFallbackCachesUnderCanonicalKey(t *testing.T) {
	tests := []struct {
		name      string
		op        cache.Op
		wantHit   cache.CacheHit
		cachedHit cache.CacheHit
	}{
		{name: "write value", op: cache.WriteValue, wantHit: cache.Hit, cachedHit: cache.Hit},
		{name: "write null", op: cache.WriteNull, wantHit: cache.Null, cachedHit: cache.Null},
		{name: "noop", op: cache.Noop, wantHit: cache.Miss, cachedHit: cache.Miss},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newFallbackCache(t)

			value, hit, err := c.SWRWithFallback(t.Context(), []string{"exact", "wildcard"}, func(context.Context) (string, string, error) {
				return "origin-value", "canonical", nil
			}, func(error) cache.Op { return tt.op })
			require.NoError(t, err)
			require.Equal(t, tt.wantHit, hit)
			require.Equal(t, "origin-value", value)

			_, hits := c.GetMany(t.Context(), []string{"exact", "wildcard", "canonical"})
			require.Equal(t, map[string]cache.CacheHit{"exact": cache.Miss, "wildcard": cache.Miss, "canonical": tt.cachedHit}, hits)
		})
	}
}

// TestSWRWithFallbackOriginErrorCachesNothing guarantees that a failed origin
// read returns its error and writes nothing, even when op asks for a write, so
// a partial result is never cached.
func TestSWRWithFallbackOriginErrorCachesNothing(t *testing.T) {
	c, _ := newFallbackCache(t)
	originErr := errors.New("origin unavailable")

	value, hit, err := c.SWRWithFallback(t.Context(), []string{"exact"}, func(context.Context) (string, string, error) {
		return "partial", "canonical", originErr
	}, func(error) cache.Op { return cache.WriteValue })
	require.ErrorIs(t, err, originErr)
	require.Equal(t, cache.Miss, hit)
	require.Empty(t, value)

	_, cached := c.Get(t.Context(), "canonical")
	require.Equal(t, cache.Miss, cached)
}
