package cache_test

import (
	"context"
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
		Metrics: nil,
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
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
		Metrics:  nil,
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
		Metrics:  nil,
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
		Metrics:  nil,
	})
	require.NoError(t, err)
	c.SetNull(context.Background(), "key")

	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Null, hit)

}
