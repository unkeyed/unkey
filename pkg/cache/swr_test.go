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
