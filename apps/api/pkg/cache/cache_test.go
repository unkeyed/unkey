package cache_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/logging"
)

func TestWriteRead(t *testing.T) {
	c := cache.New[string](cache.Config[string]{
		Fresh: time.Minute,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, error) {
			return "hello", nil
		},
		Logger: logging.New(),
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
		RefreshFromOrigin: func(ctx context.Context, id string) (string, error) {
			return "hello", nil
		},
		Logger: logging.NewNoopLogger(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	_, cacheMiss := c.Get(context.Background(), "key")
	require.False(t, cacheMiss)
}

type spyCounter struct {
	counter atomic.Int32
}

func TestRefresh(t *testing.T) {
	spy := &spyCounter{
		counter: atomic.Int32{},
	}

	c := cache.New[string](cache.Config[string]{
		Fresh: time.Second,
		Stale: time.Minute * 5,
		RefreshFromOrigin: func(ctx context.Context, id string) (string, error) {
			spy.counter.Add(1)
			return "hello", nil
		},
		Logger: logging.New(),
	})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	for i := 0; i < 10; i++ {

		_, hit := c.Get(context.Background(), "key")
		require.True(t, hit)
		time.Sleep(time.Second * 1)
	}

	require.Equal(t, int32(10), spy.counter.Load())

}
