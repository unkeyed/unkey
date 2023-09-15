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
}

func WithMetrics[T any](c cache.Cache[T], m metrics.Metrics, resource string) cache.Cache[T] {
	return &metricsMiddleware[T]{next: c, metrics: m, resource: resource}
}

func (mw *metricsMiddleware[T]) Get(ctx context.Context, key string) (T, bool) {
	start := time.Now()
	value, hit := mw.next.Get(ctx, key)
	mw.metrics.ReportCacheHit(metrics.CacheHitReport{
		Key:      key,
		Hit:      hit,
		Resource: mw.resource,
		Latency:  time.Since(start).Milliseconds(),
	})
	return value, hit
}
func (mw *metricsMiddleware[T]) Set(ctx context.Context, key string, value T) {
	mw.next.Set(ctx, key, value)

}
func (mw *metricsMiddleware[T]) Remove(ctx context.Context, key string) {

	mw.next.Remove(ctx, key)

}
