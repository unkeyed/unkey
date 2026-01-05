package middleware

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/svc/agent/pkg/cache"
	"github.com/unkeyed/unkey/svc/agent/pkg/metrics"
	"github.com/unkeyed/unkey/svc/agent/pkg/prometheus"
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

	labels := map[string]string{
		"key":      key,
		"resource": mw.resource,
		"tier":     mw.tier,
	}

	if hit == cache.Miss {
		prometheus.CacheMisses.With(labels).Inc()
	} else {
		prometheus.CacheHits.With(labels).Inc()
	}
	prometheus.CacheLatency.With(labels).Observe(time.Since(start).Seconds())

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
