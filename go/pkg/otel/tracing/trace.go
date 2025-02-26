package tracing

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var globalTracer trace.TracerProvider

func init() {
	globalTracer = noop.NewTracerProvider()
}

func SetGlobalTraceProvider(t trace.TracerProvider) {
	globalTracer = t
}

func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// nolint:spancheck // the caller will end the span
	return globalTracer.Tracer("main").Start(ctx, name, opts...)

}

func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.SetStatus(codes.Error, err.Error())
}

func GetGlobalTraceProvider() trace.TracerProvider {
	return globalTracer
}
