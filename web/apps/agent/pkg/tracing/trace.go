package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var globalTracer trace.TracerProvider

func init() {
	globalTracer = noop.NewTracerProvider()
}

func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return globalTracer.Tracer("main").Start(ctx, name, opts...)
}

func GetGlobalTraceProvider() trace.TracerProvider {
	return globalTracer
}
