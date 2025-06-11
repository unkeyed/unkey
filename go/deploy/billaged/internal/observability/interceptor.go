package observability

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the metrics for the interceptor
type Metrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
}

// NewMetrics creates new metrics
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	requestCounter, err := meter.Int64Counter(
		"rpc_server_requests_total",
		metric.WithDescription("Total number of RPC requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request counter: %w", err)
	}

	requestDuration, err := meter.Float64Histogram(
		"rpc_server_request_duration_seconds",
		metric.WithDescription("RPC request duration in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request duration histogram: %w", err)
	}

	activeRequests, err := meter.Int64UpDownCounter(
		"rpc_server_active_requests",
		metric.WithDescription("Number of active RPC requests"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create active requests counter: %w", err)
	}

	return &Metrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		activeRequests:  activeRequests,
	}, nil
}

// NewOTELInterceptor creates a new OpenTelemetry interceptor for ConnectRPC
func NewOTELInterceptor() connect.UnaryInterceptorFunc {
	tracer := otel.Tracer("billaged/rpc")
	meter := otel.Meter("billaged/rpc")

	// Create metrics
	metrics, err := NewMetrics(meter)
	if err != nil {
		// Log error but continue without metrics
		fmt.Printf("Failed to create metrics: %v\n", err)
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract procedure name
			procedure := req.Spec().Procedure

			// Start span
			ctx, span := tracer.Start(ctx, procedure,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("rpc.system", "connect_rpc"),
					attribute.String("rpc.service", req.Spec().Procedure),
					attribute.String("rpc.method", req.Spec().Procedure),
				),
			)
			defer span.End()

			// Record metrics
			if metrics != nil {
				attrs := []attribute.KeyValue{
					attribute.String("rpc.method", procedure),
				}

				// Increment active requests
				metrics.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
				defer metrics.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))
			}

			// Call the handler
			resp, err := next(ctx, req)

			// Record error and status
			statusCode := "ok"
			if err != nil {
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					statusCode = connectErr.Code().String()
					span.SetAttributes(
						attribute.String("rpc.connect.code", statusCode),
						attribute.String("rpc.connect.message", connectErr.Message()),
					)
				}

				// Record error in span
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())

				// For error sampling: create a new span that's always sampled
				if span.SpanContext().IsSampled() == false {
					_, errorSpan := tracer.Start(ctx, procedure+".error",
						trace.WithSpanKind(trace.SpanKindServer),
						trace.WithAttributes(
							attribute.String("rpc.system", "connect_rpc"),
							attribute.String("rpc.service", req.Spec().Procedure),
							attribute.String("rpc.method", req.Spec().Procedure),
							attribute.Bool("error.resampled", true),
						),
					)
					errorSpan.RecordError(err)
					errorSpan.SetStatus(codes.Error, err.Error())
					errorSpan.End()
				}
			} else {
				span.SetStatus(codes.Ok, "")
			}

			// Record metrics
			if metrics != nil {
				attrs := []attribute.KeyValue{
					attribute.String("rpc.method", procedure),
					attribute.String("rpc.status", statusCode),
				}

				// Increment request counter
				metrics.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			}

			return resp, err
		}
	}
}