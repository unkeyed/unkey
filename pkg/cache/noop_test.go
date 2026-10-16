package cache_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
)

// TestNoopCacheReportsMisses guarantees that a disabled cache reports every
// read as a miss, never as a cached null, because callers treat null as "the
// row does not exist".
func TestNoopCacheReportsMisses(t *testing.T) {
	c := cache.NewNoopCache[string, string]()
	ctx := t.Context()
	keys := []string{"a", "b"}
	allMiss := map[string]cache.CacheHit{"a": cache.Miss, "b": cache.Miss}
	op := func(error) cache.Op { return cache.WriteValue }

	c.Set(ctx, "a", "value")

	_, hit := c.Get(ctx, "a")
	require.Equal(t, cache.Miss, hit)

	values, hits := c.GetMany(ctx, keys)
	require.NotNil(t, values)
	require.Empty(t, values)
	require.Equal(t, allMiss, hits)

	_, hit, err := c.SWR(ctx, "a", func(context.Context) (string, error) { return "value", nil }, op)
	require.NoError(t, err)
	require.Equal(t, cache.Miss, hit)

	values, hits, err = c.SWRMany(ctx, keys, func(context.Context, []string) (map[string]string, error) {
		return map[string]string{"a": "value"}, nil
	}, op)
	require.NoError(t, err)
	require.NotNil(t, values)
	require.Empty(t, values)
	require.Equal(t, allMiss, hits)

	_, hit, err = c.SWRWithFallback(ctx, keys, func(context.Context) (string, string, error) { return "value", "a", nil }, op)
	require.NoError(t, err)
	require.Equal(t, cache.Miss, hit)

	require.Equal(t, "noop", c.Name())
}
