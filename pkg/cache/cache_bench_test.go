package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
)

// BenchmarkGet measures an in-memory read, which records a read counter and
// an operation-duration histogram on every call. Metric overhead on this
// path is paid on every key verification.
func BenchmarkGet(b *testing.B) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  1_000,
		Resource: "bench",
		Clock:    clock.New(),
	})
	require.NoError(b, err)
	ctx := context.Background()
	c.Set(ctx, "key", "value")

	b.Run("hit", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			c.Get(ctx, "key")
		}
	})

	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			c.Get(ctx, "missing")
		}
	})

	b.Run("hit_parallel", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				c.Get(ctx, "key")
			}
		})
	})
}
