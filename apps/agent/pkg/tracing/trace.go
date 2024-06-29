package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var globalTracer trace.Tracer

func init() {
	globalTracer = noop.NewTracerProvider().Tracer("noop")
}

func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return globalTracer.Start(ctx, name, opts...)
}
