package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/chronark/unkey/apps/api/pkg/cache"
)

func TestWriteRead(t *testing.T) {
	c := cache.NewInMemoryCache[string]()
	c.Set(context.Background(), "key", "value", time.Now().Add(time.Minute))
	value, cacheHit := c.Get(context.Background(), "key", false)
	require.True(t, cacheHit)
	require.Equal(t, "value", value)
}

func TestEviction(t *testing.T) {
	c := cache.NewInMemoryCache[string]()
	c.Set(context.Background(), "key", "value", time.Now().Add(time.Second))
	time.Sleep(time.Second * 2)
	_, cacheMiss := c.Get(context.Background(), "key", false)
	require.False(t, cacheMiss)
}
