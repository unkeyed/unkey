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

func TestWriteRead(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize: 10_000,

		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Resource: "test", Clock: clock.New(),
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	value, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "value", value)
	require.Equal(t, "test", c.Name())
}

func TestClose(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	c.Set(context.Background(), "key", "value")

	c.Close()
	// Close must be idempotent.
	c.Close()

	// Cached data stays readable after Close.
	value, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "value", value)
}

func TestEviction(t *testing.T) {

	clk := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		MaxSize: 10_000,

		Fresh:    time.Second,
		Stale:    time.Second,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	clk.Tick(2 * time.Second)
	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Miss, hit)
}

func TestRefresh(t *testing.T) {

	clk := clock.NewTestClock()

	// count how many times we refreshed from origin
	refreshedFromOrigin := atomic.Int32{}

	c, err := cache.New(cache.Config[string, string]{
		MaxSize: 10_000,

		Fresh:    time.Second * 2,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	clk.Tick(time.Second)

	for i := 0; i < 10; i++ {
		_, hit := c.Get(context.Background(), "key")
		require.Equal(t, cache.Hit, hit)
		clk.Tick(time.Second)
	}
	require.LessOrEqual(t, refreshedFromOrigin.Load(), int32(5))

}

func TestNull(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Second * 1,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	c.SetNull(context.Background(), "key")

	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Null, hit)

}

func TestNewRejectsInvalidConfig(t *testing.T) {
	valid := cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clock.New(),
	}
	tests := []struct {
		name    string
		mutate  func(*cache.Config[string, string])
		wantErr string
	}{
		{name: "missing clock", mutate: func(c *cache.Config[string, string]) { c.Clock = nil }, wantErr: "clock is required"},
		{name: "zero max size", mutate: func(c *cache.Config[string, string]) { c.MaxSize = 0 }, wantErr: "capacity"},
		{name: "zero stale time", mutate: func(c *cache.Config[string, string]) { c.Stale = 0 }, wantErr: "ttl"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := valid
			tt.mutate(&config)
			c, err := cache.New(config)
			require.ErrorContains(t, err, tt.wantErr)
			require.Nil(t, c)
		})
	}
}

// TestMaxSizeBoundsEntries guarantees that a cache sized for n entries keeps n
// entries, and that writing many more keys evicts entries instead of growing
// memory without limit.
func TestMaxSizeBoundsEntries(t *testing.T) {
	const maxSize = 10
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  maxSize,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	keys := make([]string, 0, 100*maxSize)
	for i := range maxSize {
		keys = append(keys, fmt.Sprint(i))
		c.Set(t.Context(), keys[i], fmt.Sprint("value-", i))
	}
	for i, key := range keys {
		value, hit := c.Get(t.Context(), key)
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, fmt.Sprint("value-", i), value)
	}

	for i := maxSize; i < cap(keys); i++ {
		keys = append(keys, fmt.Sprint(i))
		c.Set(t.Context(), keys[i], "value")
	}
	require.Eventually(t, func() bool {
		values, _ := c.GetMany(t.Context(), keys)
		return len(values) <= maxSize
	}, 5*time.Second, time.Millisecond)
}

func TestGetMissesUnknownKey(t *testing.T) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	value, hit := c.Get(t.Context(), "unknown")
	require.Equal(t, cache.Miss, hit, "an unknown key must not look like a cached null")
	require.Empty(t, value)
}

func TestRemoveDeletesEveryKey(t *testing.T) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	c.SetMany(t.Context(), map[string]string{"a": "va", "b": "vb", "c": "vc"})

	c.Remove(t.Context(), "a", "b")

	_, hits := c.GetMany(t.Context(), []string{"a", "b", "c"})
	require.Equal(t, map[string]cache.CacheHit{"a": cache.Miss, "b": cache.Miss, "c": cache.Hit}, hits)
}

// TestDumpRestoreKeepsServableEntries guarantees that restoring a dump, as a
// restarted process does, brings back every entry that may still be served,
// including stale ones, and drops entries that are past their stale time.
func TestDumpRestoreKeepsServableEntries(t *testing.T) {
	clk := clock.NewTestClock()
	config := cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clk,
	}
	source, err := cache.New(config)
	require.NoError(t, err)
	t.Cleanup(source.Close)

	source.Set(t.Context(), "expired", "expired-value")
	clk.Tick(30 * time.Minute)
	source.Set(t.Context(), "stale", "stale-value")
	clk.Tick(31 * time.Minute)
	source.Set(t.Context(), "fresh", "fresh-value")

	dump, err := source.Dump(t.Context())
	require.NoError(t, err)

	restored, err := cache.New(config)
	require.NoError(t, err)
	t.Cleanup(restored.Close)
	require.NoError(t, restored.Restore(t.Context(), dump))

	values, hits := restored.GetMany(t.Context(), []string{"expired", "stale", "fresh"})
	require.Equal(t, map[string]string{"stale": "stale-value", "fresh": "fresh-value"}, values)
	require.Equal(t, map[string]cache.CacheHit{"expired": cache.Miss, "stale": cache.Hit, "fresh": cache.Hit}, hits)
}

func TestDumpFailsForUnserializableValues(t *testing.T) {
	c, err := cache.New(cache.Config[string, func()]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	c.Set(t.Context(), "key", func() {})

	dump, err := c.Dump(t.Context())
	require.Error(t, err)
	require.Nil(t, dump)
}

func TestRestoreRejectsMalformedDump(t *testing.T) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	require.Error(t, c.Restore(t.Context(), []byte("not a cache dump")))
}
