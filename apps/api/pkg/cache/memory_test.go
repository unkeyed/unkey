package cache_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
)

func TestWriteRead(t *testing.T) {
	c := cache.NewInMemoryCache[string](cache.Config{Ttl: time.Minute, Tracer: tracing.NewNoop()})
	c.Set(context.Background(), "key", "value")
	value, cacheHit := c.Get(context.Background(), "key")
	require.True(t, cacheHit)
	require.Equal(t, "value", value)

}

func TestEviction(t *testing.T) {
	c := cache.NewInMemoryCache[string](cache.Config{Ttl: time.Second, Tracer: tracing.NewNoop()})
	c.Set(context.Background(), "key", "value")
	time.Sleep(time.Second * 2)
	_, cacheMiss := c.Get(context.Background(), "key")
	require.False(t, cacheMiss)
}

func TestClear(t *testing.T) {
	c := cache.NewInMemoryCache[int](cache.Config{Ttl: 0, Tracer: tracing.NewNoop()})
	for i := 0; i < 200; i++ {
		c.Set(context.Background(), fmt.Sprintf("%d", i), i)
	}

	require.Equal(t, 200, c.Size())
	c.Clear()
	require.Equal(t, 0, c.Size())
}
