package middleware

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/"github.com/unkeyed/unkey/go/pkg/otel""
	"go.opentelemetry.io/otel/attribute"
)

type tracingMiddleware[K comparable, V any] struct {
	next cache.Cache[K, V]
}

func WithTracing[K comparable, V any](c cache.Cache[K, V]) cache.Cache[K, V] {
	return &tracingMiddleware[K, V]{next: c}
}

func (mw *tracingMiddleware[K, V]) Get(ctx context.Context, key K) (V, cache.CacheHit) {
	ctx, span := tracing.Start(ctx, "cache.Get")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprintf("%+v", key)))

	value, hit := mw.next.Get(ctx, key)
	span.SetAttributes(
		attribute.Bool("hit", hit != cache.Miss),
	)
	return value, hit
}
func (mw *tracingMiddleware[K, V]) Set(ctx context.Context, key K, value V) {
	ctx, span := tracing.Start(ctx, "cache.Set")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprintf("%+v", key)))

	mw.next.Set(ctx, key, value)

}
func (mw *tracingMiddleware[K, V]) SetNull(ctx context.Context, key K) {
	ctx, span := tracing.Start(ctx, "cache.SetNull")
	defer span.End()

	span.SetAttributes(attribute.String("key", fmt.Sprintf("%+v", key)))
	mw.next.SetNull(ctx, key)

}
func (mw *tracingMiddleware[K, V]) Remove(ctx context.Context, key K) {
	ctx, span := tracing.Start(ctx, "cache.Remove")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprintf("%+v", key)))

	mw.next.Remove(ctx, key)

}

func (mw *tracingMiddleware[K, V]) Dump(ctx context.Context) ([]byte, error) {
	ctx, span := tracing.Start(ctx, "cache.Dump")
	defer span.End()

	b, err := mw.next.Dump(ctx)
	if err != nil {
		tracing.RecordError(span, err)
	}
	// nolint:wrapcheck
	return b, err
}

func (mw *tracingMiddleware[K, V]) Restore(ctx context.Context, data []byte) error {
	ctx, span := tracing.Start(ctx, "cache.Restore")
	defer span.End()

	err := mw.next.Restore(ctx, data)
	if err != nil {
		tracing.RecordError(span, err)
	}
	// nolint:wrapcheck
	return err
}

func (mw *tracingMiddleware[K, V]) Clear(ctx context.Context) {
	ctx, span := tracing.Start(ctx, "cache.Clear")
	defer span.End()

	mw.next.Clear(ctx)
}

func (mw *tracingMiddleware[K, V]) SWR(ctx context.Context, key K, refreshFromOrigin func(ctx context.Context) (V, error), translateError func(err error) cache.CacheHit) (V, error) {
	ctx, span := tracing.Start(ctx, "cache.SWR")
	defer span.End()
	span.SetAttributes(attribute.String("key", fmt.Sprintf("%+v", key)))

	value, err := mw.next.SWR(ctx, key, refreshFromOrigin, translateError)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return value, err

}
