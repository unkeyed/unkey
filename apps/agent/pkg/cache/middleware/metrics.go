package middleware

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
)

type metricsMiddleware[T any] struct {
	next     cache.Cache[T]
	metrics  metrics.Metrics
	resource string
	tier     string
}

func WithMetrics[T any](c cache.Cache[T], m metrics.Metrics, resource string, tier string) cache.Cache[T] {
	return &metricsMiddleware[T]{next: c, metrics: m, resource: resource, tier: tier}
}

func (mw *metricsMiddleware[T]) Get(ctx context.Context, key string) (T, cache.CacheHit) {
	start := time.Now()
	value, hit := mw.next.Get(ctx, key)
	mw.metrics.Record(metrics.CacheHit{
		Key:      key,
		Hit:      hit != cache.Miss,
		Resource: mw.resource,
		Latency:  time.Since(start).Milliseconds(),
		Tier:     mw.tier,
	})
	return value, hit
}
func (mw *metricsMiddleware[T]) Set(ctx context.Context, key string, value T) {
	mw.next.Set(ctx, key, value)

}
func (mw *metricsMiddleware[T]) SetNull(ctx context.Context, key string) {
	mw.next.SetNull(ctx, key)

}
func (mw *metricsMiddleware[T]) Remove(ctx context.Context, key string) {

	mw.next.Remove(ctx, key)

}

func (mw *metricsMiddleware[T]) Dump(ctx context.Context) ([]byte, error) {
	return mw.next.Dump(ctx)
}

func (mw *metricsMiddleware[T]) Restore(ctx context.Context, data []byte) error {
	return mw.next.Restore(ctx, data)
}

func (mw *metricsMiddleware[T]) Clear(ctx context.Context) {
	mw.next.Clear(ctx)
}
