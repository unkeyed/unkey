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
	c := cache.New[string](cache.Config[string]{
		Fresh: time.Minute,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool, error) {
			return "hello", true, nil
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	value, cacheHit := c.Get(context.Background(), "key")
	require.True(t, cacheHit)
	require.Equal(t, "value", value)
}

func TestEviction(t *testing.T) {
	c := cache.New[string](cache.Config[string]{
		Fresh: time.Second,
		Stale: time.Second,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool, error) {
			return "hello", true, nil
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	_, cacheMiss := c.Get(context.Background(), "key")
	require.False(t, cacheMiss)
}

func TestRefresh(t *testing.T) {

	// count how many times we refreshed from origin
	refreshedFromOrigin := atomic.Int32{}

	c := cache.New[string](cache.Config[string]{
		Fresh: time.Second * 2,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, bool, error) {
			refreshedFromOrigin.Add(1)
			return "hello", true, nil
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	for i := 0; i < 10; i++ {
		_, hit := c.Get(context.Background(), "key")
		require.True(t, hit)
		time.Sleep(time.Second * 1)
	}

	require.Equal(t, int32(5), refreshedFromOrigin.Load())

}
