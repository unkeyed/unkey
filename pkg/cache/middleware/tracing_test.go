package middleware_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/cache/middleware"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type recordedSpan struct {
	name   string
	parent string
	attrs  map[string]any
	status codes.Code
}

// TestTracingMiddleware guarantees that wrapping a cache with tracing returns
// exactly what the wrapped cache returns, and that every operation records a
// span with the attributes operators filter on, with origin reads nested under
// the cache operation that started them and failures marked as errors.
func TestTracingMiddleware(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := tracing.GetGlobalTraceProvider()
	tracing.SetGlobalTraceProvider(provider)
	t.Cleanup(func() {
		tracing.SetGlobalTraceProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})

	inner, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  100,
		Resource: "traced",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	c := middleware.WithTracing(inner)
	t.Cleanup(c.Close)
	ctx := t.Context()
	originErr := errors.New("origin unavailable")
	op := func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	}

	c.Set(ctx, "a", "va")
	c.SetMany(ctx, map[string]string{"b": "vb"})
	c.SetNull(ctx, "n")
	c.SetNullMany(ctx, []string{"m"})

	value, hit := c.Get(ctx, "a")
	require.Equal(t, "va", value)
	require.Equal(t, cache.Hit, hit)
	_, hit = c.Get(ctx, "missing")
	require.Equal(t, cache.Miss, hit)

	values, hits := c.GetMany(ctx, []string{"b", "m", "missing"})
	require.Equal(t, map[string]string{"b": "vb", "m": ""}, values)
	require.Equal(t, map[string]cache.CacheHit{"b": cache.Hit, "m": cache.Null, "missing": cache.Miss}, hits)

	value, hit, err = c.SWR(ctx, "c", func(context.Context) (string, error) { return "vc", nil }, op)
	require.NoError(t, err)
	require.Equal(t, "vc", value)
	require.Equal(t, cache.Hit, hit)
	_, hit, err = c.SWR(ctx, "e", func(context.Context) (string, error) { return "", originErr }, op)
	require.ErrorIs(t, err, originErr)
	require.Equal(t, cache.Miss, hit)

	values, hits, err = c.SWRMany(ctx, []string{"a", "d"}, func(_ context.Context, keys []string) (map[string]string, error) {
		require.Equal(t, []string{"d"}, keys)
		return map[string]string{"d": "vd"}, nil
	}, op)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"a": "va", "d": "vd"}, values)
	require.Equal(t, map[string]cache.CacheHit{"a": cache.Hit, "d": cache.Hit}, hits)
	_, hits, err = c.SWRMany(ctx, []string{"x"}, func(context.Context, []string) (map[string]string, error) {
		return nil, originErr
	}, op)
	require.ErrorIs(t, err, originErr)
	require.Equal(t, map[string]cache.CacheHit{"x": cache.Miss}, hits)

	value, hit, err = c.SWRWithFallback(ctx, []string{"f", "*.f"}, func(context.Context) (string, string, error) {
		return "vf", "*.f", nil
	}, op)
	require.NoError(t, err)
	require.Equal(t, "vf", value)
	require.Equal(t, cache.Hit, hit)
	_, hit, err = c.SWRWithFallback(ctx, []string{"h"}, func(context.Context) (string, string, error) {
		return "", "", originErr
	}, op)
	require.ErrorIs(t, err, originErr)
	require.Equal(t, cache.Miss, hit)

	c.Remove(ctx, "a", "b")
	_, hit = inner.Get(ctx, "a")
	require.Equal(t, cache.Miss, hit)

	dump, err := c.Dump(ctx)
	require.NoError(t, err)
	c.Clear(ctx)
	_, hit = inner.Get(ctx, "c")
	require.Equal(t, cache.Miss, hit)
	require.NoError(t, c.Restore(ctx, dump))
	_, hit = inner.Get(ctx, "c")
	require.Equal(t, cache.Hit, hit)
	require.Error(t, c.Restore(ctx, []byte("not a cache dump")))

	unserializable, err := cache.New(cache.Config[string, func()]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "unserializable",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(unserializable.Close)
	unserializable.Set(ctx, "key", func() {})
	_, err = middleware.WithTracing(unserializable).Dump(ctx)
	require.Error(t, err)

	require.Equal(t, "traced", c.Name())

	none := map[string]any{}
	require.Equal(t, []recordedSpan{
		{name: "cache.Set", attrs: map[string]any{"key": "a"}},
		{name: "cache.SetMany", attrs: map[string]any{"count": int64(1)}},
		{name: "cache.SetNull", attrs: map[string]any{"key": "n"}},
		{name: "cache.SetNullMany", attrs: map[string]any{"count": int64(1)}},
		{name: "cache.Get", attrs: map[string]any{"key": "a", "hit": true}},
		{name: "cache.Get", attrs: map[string]any{"key": "missing", "hit": false}},
		{name: "cache.GetMany", attrs: map[string]any{"count": int64(3), "hits": int64(2), "misses": int64(1)}},
		{name: "refreshFromOrigin", parent: "cache.SWR", attrs: none},
		{name: "cache.SWR", attrs: map[string]any{"key": "c"}},
		{name: "refreshFromOrigin", parent: "cache.SWR", attrs: none},
		{name: "cache.SWR", attrs: map[string]any{"key": "e"}, status: codes.Error},
		{name: "refreshFromOrigin", parent: "cache.SWRMany", attrs: map[string]any{"keys": int64(1)}},
		{name: "cache.SWRMany", attrs: map[string]any{"count": int64(2), "hits": int64(2), "misses": int64(0)}},
		{name: "refreshFromOrigin", parent: "cache.SWRMany", attrs: map[string]any{"keys": int64(1)}},
		{name: "cache.SWRMany", attrs: map[string]any{"count": int64(1), "hits": int64(0), "misses": int64(1)}, status: codes.Error},
		{name: "refreshFromOrigin", parent: "cache.SWRWithFallback", attrs: none},
		{name: "cache.SWRWithFallback", attrs: map[string]any{"candidates": int64(2), "hit": true}},
		{name: "refreshFromOrigin", parent: "cache.SWRWithFallback", attrs: none},
		{name: "cache.SWRWithFallback", attrs: map[string]any{"candidates": int64(1), "hit": false}, status: codes.Error},
		{name: "cache.Remove", attrs: map[string]any{"keys": "[a b]", "count": int64(2)}},
		{name: "cache.Dump", attrs: none},
		{name: "cache.Clear", attrs: none},
		{name: "cache.Restore", attrs: none},
		{name: "cache.Restore", attrs: none, status: codes.Error},
		{name: "cache.Dump", attrs: none, status: codes.Error},
	}, recordedSpans(recorder))
}

func recordedSpans(recorder *tracetest.SpanRecorder) []recordedSpan {
	ended := recorder.Ended()
	names := make(map[string]string, len(ended))
	for _, span := range ended {
		names[span.SpanContext().SpanID().String()] = span.Name()
	}
	spans := make([]recordedSpan, 0, len(ended))
	for _, span := range ended {
		attrs := make(map[string]any, len(span.Attributes()))
		for _, attr := range span.Attributes() {
			attrs[string(attr.Key)] = attr.Value.AsInterface()
		}
		spans = append(spans, recordedSpan{
			name:   span.Name(),
			parent: names[span.Parent().SpanID().String()],
			attrs:  attrs,
			status: span.Status().Code,
		})
	}
	return spans
}
