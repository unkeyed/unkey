package cache_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestSWR_MaxAge(t *testing.T) {
	newCache := func(t *testing.T) (cache.Cache[string, string], *clock.TestClock) {
		t.Helper()
		clk := clock.NewTestClock()
		c, err := cache.New(cache.Config[string, string]{
			Fresh: time.Minute, Stale: 5 * time.Minute, MaxSize: 100, Resource: "test", Clock: clk,
		})
		require.NoError(t, err)
		t.Cleanup(c.Close)
		return c, clk
	}
	var loads atomic.Int64
	origin := func(context.Context) (string, error) {
		return fmt.Sprintf("v%d", loads.Add(1)), nil
	}
	write := func(error) cache.Op { return cache.WriteValue }
	read := func(t *testing.T, c cache.Cache[string, string], opts ...cache.SWROption) string {
		t.Helper()
		value, _, err := c.SWR(t.Context(), "key", origin, write, opts...)
		require.NoError(t, err)
		return value
	}

	t.Run("max age fresh alone waits for the origin once the value is that old", func(t *testing.T) {
		c, clk := newCache(t)
		loads.Store(0)
		require.Equal(t, "v1", read(t, c))

		clk.Tick(100*time.Millisecond - time.Nanosecond)
		require.Equal(t, "v1", read(t, c, cache.MaxAgeFresh(100*time.Millisecond)))

		clk.Tick(time.Nanosecond)
		require.Equal(t, "v1", read(t, c), "a read without options still uses the cache's limits")
		require.Equal(t, "v2", read(t, c, cache.MaxAgeFresh(100*time.Millisecond)))
		require.Equal(t, "v2", read(t, c), "the blocking load refreshes the value for every reader")
	})

	t.Run("max age stale serves and revalidates until it is reached", func(t *testing.T) {
		c, clk := newCache(t)
		loads.Store(0)
		require.Equal(t, "v1", read(t, c))
		opts := []cache.SWROption{cache.MaxAgeFresh(100 * time.Millisecond), cache.MaxAgeStale(200 * time.Millisecond)}

		clk.Tick(150 * time.Millisecond)
		require.Equal(t, "v1", read(t, c, opts...))
		require.Eventually(t, func() bool { return loads.Load() == 2 }, 5*time.Second, 10*time.Millisecond)

		clk.Tick(200 * time.Millisecond)
		require.Eventually(t, func() bool { return read(t, c) == "v2" }, 5*time.Second, 10*time.Millisecond)
		require.Equal(t, "v3", read(t, c, opts...))
	})

	t.Run("options never raise the cache's limits", func(t *testing.T) {
		c, clk := newCache(t)
		loads.Store(0)
		require.Equal(t, "v1", read(t, c))

		clk.Tick(5 * time.Minute)
		require.Equal(t, "v2", read(t, c, cache.MaxAgeFresh(time.Hour), cache.MaxAgeStale(time.Hour)))
	})
}
