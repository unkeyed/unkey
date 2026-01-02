package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// globalTracer holds the trace provider for the application.
// It's initialized to a no-op implementation but should be replaced
// with a real implementation during application startup.
var globalTracer trace.TracerProvider

// init initializes the global tracer to a no-op implementation.
// This ensures that tracing functions can be called safely before
// any specific tracer is configured.
func init() {
	globalTracer = noop.NewTracerProvider()
	otel.SetTextMapPropagator(propagation.TraceContext{})
}

// SetGlobalTraceProvider sets the global trace provider for the application.
// This should be called early in the application startup, typically during
// the initialization of the observability systems.
//
// Example:
//
//	// During application initialization
//	traceExporter, _ := otlptrace.New(ctx, otlptracehttp.NewClient())
//	traceProvider := trace.NewTracerProvider(trace.WithBatcher(traceExporter))
//	tracing.SetGlobalTraceProvider(traceProvider)
func SetGlobalTraceProvider(t trace.TracerProvider) {
	globalTracer = t
}

// Start creates a new span and context.
// It uses the global tracer provider to create spans with the specified name.
// Additional trace options can be provided for customization.
//
// The caller is responsible for ending the span with span.End().
//
// Example:
//
//	func ProcessOrder(ctx context.Context, orderID string) error {
//	    ctx, span := tracing.Start(ctx, "orders.ProcessOrder")
//	    defer span.End()
//
//	    // Span now tracks this function execution
//	    // Add more details to the span as needed
//	    span.SetAttributes(attribute.String("order_id", orderID))
//
//	    // Proceed with actual business logic
//	    // ...
//	}
func Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	// nolint:spancheck // the caller will end the span
	return globalTracer.Tracer("main").Start(ctx, name, opts...)
}

// RecordError marks a span as having encountered an error.
// If the error is nil, this function does nothing.
// This should be called whenever an error occurs within a traced operation
// to ensure that errors are properly recorded in the tracing system.
//
// Example:
//
//	ctx, span := tracing.Start(ctx, "database.Query")
//	defer span.End()
//
//	result, err := db.Query(ctx, "SELECT * FROM users WHERE id = ?", userID)
//	if err != nil {
//	    tracing.RecordError(span, err)
//	    return nil, err
//	}
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.SetStatus(codes.Error, err.Error())
}

// GetGlobalTraceProvider returns the current global trace provider.
// This can be useful for passing the trace provider to other systems
// that need to create their own tracers.
//
// Example:
//
//	// Setting up a Connect RPC client with tracing
//	interceptor, err := otelconnect.NewInterceptor(
//	    otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()),
//	)
func GetGlobalTraceProvider() trace.TracerProvider {
	return globalTracer
}
