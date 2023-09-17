package middleware

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware[T any] struct {
	next   cache.Cache[T]
	tracer tracing.Tracer
}

func WithTracing[T any](c cache.Cache[T], t tracing.Tracer) cache.Cache[T] {
	return &tracingMiddleware[T]{next: c, tracer: t}
}

func (mw *tracingMiddleware[T]) Get(ctx context.Context, key string) (T, cache.CacheHit) {
	ctx, span := mw.tracer.Start(ctx, "cache.Get", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	value, hit := mw.next.Get(ctx, key)
	span.SetAttributes(
		attribute.Bool("hit", hit != cache.Miss),
	)
	return value, hit
}
func (mw *tracingMiddleware[T]) Set(ctx context.Context, key string, value T) {
	ctx, span := mw.tracer.Start(ctx, "cache.Set", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	mw.next.Set(ctx, key, value)

}
func (mw *tracingMiddleware[T]) SetNull(ctx context.Context, key string) {
	ctx, span := mw.tracer.Start(ctx, "cache.SetNull", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	mw.next.SetNull(ctx, key)

}
func (mw *tracingMiddleware[T]) Remove(ctx context.Context, key string) {
	ctx, span := mw.tracer.Start(ctx, "cache.Remove", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	mw.next.Remove(ctx, key)

}
