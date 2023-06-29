package cache_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/chronark/unkey/apps/api/pkg/cache"
)

func TestWriteRead(t *testing.T) {
	c := cache.NewInMemoryCache[string](time.Second)
	c.Set("key", "value")
	value, cacheHit := c.Get("key")
	require.True(t, cacheHit)
	require.Equal(t, "value", value)

}

func TestEviction(t *testing.T) {
	c := cache.NewInMemoryCache[string](time.Second)
	c.Set("key", "value")
	time.Sleep(time.Second * 2)
	_, cacheMiss := c.Get("key")
	require.False(t, cacheMiss)
}

func TestClear(t *testing.T) {
	c := cache.NewInMemoryCache[int](0)
	for i := 0; i < 200; i++ {
		c.Set(fmt.Sprintf("%d", i), i)
	}

	require.Equal(t, 200, c.Size())
	c.Clear()
	require.Equal(t, 0, c.Size())
}
