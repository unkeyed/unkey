package interceptors

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// NewClientTracePropagationInterceptor creates a ConnectRPC interceptor that propagates
// OpenTelemetry trace context to outgoing RPC requests. This ensures distributed traces
// span across service boundaries.
//
// AIDEV-NOTE: This interceptor is essential for distributed tracing in microservices.
// It must be the first interceptor in the chain to ensure trace context is available
// for all subsequent interceptors and the actual RPC call.
func NewClientTracePropagationInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Use the global propagator to inject trace context into headers
			propagator := otel.GetTextMapPropagator()
			propagator.Inject(ctx, propagation.HeaderCarrier(req.Header()))

			// Log trace propagation for debugging
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				logger.LogAttrs(ctx, slog.LevelDebug, "propagating trace context",
					slog.String("trace_id", span.SpanContext().TraceID().String()),
					slog.String("span_id", span.SpanContext().SpanID().String()),
					slog.String("procedure", req.Spec().Procedure),
					slog.Bool("sampled", span.SpanContext().IsSampled()),
				)
			}

			return next(ctx, req)
		}
	}
}

// NewClientMetricsInterceptor creates a ConnectRPC interceptor for client-side metrics.
// It creates spans for outgoing RPC calls and tracks their duration and status.
//
// AIDEV-NOTE: This creates CLIENT spans, which are different from SERVER spans.
// The trace propagation interceptor ensures these spans are properly linked to
// the parent trace.
func NewClientMetricsInterceptor(serviceName string, logger *slog.Logger) connect.UnaryInterceptorFunc {
	tracer := otel.Tracer(serviceName)

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract procedure info
			procedure := req.Spec().Procedure

			// Create a client span
			ctx, span := tracer.Start(ctx, procedure,
				trace.WithSpanKind(trace.SpanKindClient),
				trace.WithAttributes(
					attribute.String("rpc.system", "connect_rpc"),
					attribute.String("rpc.service", serviceName),
					attribute.String("rpc.method", procedure),
				),
			)
			defer span.End()

			// Execute the RPC call
			resp, err := next(ctx, req)

			// Record the result
			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return resp, err
		}
	}
}

// NewDefaultClientInterceptors returns a set of default interceptors for RPC clients.
// The interceptors are returned in the correct order for optimal functionality:
// 1. Trace propagation - ensures trace context is in headers
// 2. Client metrics - creates client spans
// 3. Tenant forwarding - adds tenant headers
//
// AIDEV-NOTE: The order matters! Trace propagation must happen first so that
// the client metrics interceptor can create spans that are properly linked
// to the parent trace.
func NewDefaultClientInterceptors(serviceName string, logger *slog.Logger) []connect.UnaryInterceptorFunc {
	return []connect.UnaryInterceptorFunc{
		NewClientTracePropagationInterceptor(logger),
		NewClientMetricsInterceptor(serviceName, logger),
	}
}
