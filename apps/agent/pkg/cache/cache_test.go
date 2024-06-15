package cache_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

func TestWriteRead(t *testing.T) {
	c := cache.NewMemory[string](cache.Config[string]{
		Fresh: time.Minute,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool) {
			return "hello", true
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	value, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "value", value)
}

func TestEviction(t *testing.T) {
	c := cache.NewMemory[string](cache.Config[string]{
		Fresh: time.Second,
		Stale: time.Second,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool) {
			return "hello", true
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Miss, hit)
}

func TestRefresh(t *testing.T) {

	// count how many times we refreshed from origin
	refreshedFromOrigin := atomic.Int32{}

	c := cache.NewMemory[string](cache.Config[string]{
		Fresh: time.Second * 2,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool) {
			refreshedFromOrigin.Add(1)
			return "hello", true
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 1)
	for i := 0; i < 10; i++ {
		_, hit := c.Get(context.Background(), "key")
		require.Equal(t, cache.Hit, hit)
		time.Sleep(time.Second * 1)
	}

	time.Sleep(5 * time.Second)

	require.Equal(t, int32(5), refreshedFromOrigin.Load())

}

func TestNull(t *testing.T) {

	c := cache.NewMemory[string](cache.Config[string]{
		Fresh:  time.Second * 1,
		Stale:  time.Minute * 5,
		Logger: logging.NewNoopLogger(),
	})
	c.SetNull(context.Background(), "key")

	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Null, hit)

}
