package middleware

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type tracingMiddleware[T any] struct {
	next cache.Cache[T]
}

func WithTracing[T any](c cache.Cache[T]) cache.Cache[T] {
	return &tracingMiddleware[T]{next: c}
}

func (mw *tracingMiddleware[T]) Get(ctx context.Context, key string) (T, cache.CacheHit) {
	ctx, span := tracing.Start(ctx, "cache.Get")
	defer span.End()
	span.SetAttributes(attribute.String("key", key))

	value, hit := mw.next.Get(ctx, key)
	span.SetAttributes(
		attribute.Bool("hit", hit != cache.Miss),
	)
	return value, hit
}
func (mw *tracingMiddleware[T]) Set(ctx context.Context, key string, value T) {
	ctx, span := tracing.Start(ctx, "cache.Set")
	defer span.End()
	span.SetAttributes(attribute.String("key", key))

	mw.next.Set(ctx, key, value)

}
func (mw *tracingMiddleware[T]) SetNull(ctx context.Context, key string) {
	ctx, span := tracing.Start(ctx, "cache.SetNull")
	defer span.End()

	span.SetAttributes(attribute.String("key", key))
	mw.next.SetNull(ctx, key)

}
func (mw *tracingMiddleware[T]) Remove(ctx context.Context, key string) {
	ctx, span := tracing.Start(ctx, "cache.Remove")
	defer span.End()
	span.SetAttributes(attribute.String("key", key))

	mw.next.Remove(ctx, key)

}

func (mw *tracingMiddleware[T]) Dump(ctx context.Context) ([]byte, error) {
	ctx, span := tracing.Start(ctx, "cache.Dump")
	defer span.End()

	b, err := mw.next.Dump(ctx)
	if err != nil {
		tracing.RecordError(span, err)
	}
	// nolint:wrapcheck
	return b, err
}

func (mw *tracingMiddleware[T]) Restore(ctx context.Context, data []byte) error {
	ctx, span := tracing.Start(ctx, "cache.Restore")
	defer span.End()

	err := mw.next.Restore(ctx, data)
	if err != nil {
		tracing.RecordError(span, err)
	}
	// nolint:wrapcheck
	return err
}

func (mw *tracingMiddleware[T]) Clear(ctx context.Context) {
	ctx, span := tracing.Start(ctx, "cache.Clear")
	defer span.End()

	mw.next.Clear(ctx)
}

func (mw *tracingMiddleware[T]) SWR(ctx context.Context, key string) (T, bool) {
	ctx, span := tracing.Start(ctx, "cache.SWR")
	defer span.End()
	span.SetAttributes(attribute.String("key", key))

	value, found := mw.next.SWR(ctx, key)
	span.SetAttributes(attribute.Bool("found", found))
	return value, found

}
