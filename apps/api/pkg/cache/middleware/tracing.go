package middleware

import (
	"context"
	"time"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"github.com/chronark/unkey/apps/api/pkg/tracing"
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

func (mw *tracingMiddleware[T]) Get(ctx context.Context, key string, refresh bool) (T, bool) {
	ctx, span := mw.tracer.Start(ctx, "cache.get", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	value, found := mw.next.Get(ctx, key, refresh)
	span.SetAttributes(
		attribute.Bool("found", found),
	)
	return value, found
}
func (mw *tracingMiddleware[T]) Set(ctx context.Context, key string, value T, exp time.Time) {
	ctx, span := mw.tracer.Start(ctx, "cache.set", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	mw.next.Set(ctx, key, value, exp)

}
func (mw *tracingMiddleware[T]) Remove(ctx context.Context, key string) {
	ctx, span := mw.tracer.Start(ctx, "cache.remove", trace.WithAttributes(attribute.String("key", key)))
	defer span.End()

	mw.next.Remove(ctx, key)

}
