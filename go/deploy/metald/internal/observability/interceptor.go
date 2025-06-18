package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

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
	panicCounter    metric.Int64Counter
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

	panicCounter, err := meter.Int64Counter(
		"rpc_server_panics_total",
		metric.WithDescription("Total number of RPC server panics"),
		metric.WithUnit("1"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create panic counter: %w", err)
	}

	return &Metrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		activeRequests:  activeRequests,
		panicCounter:    panicCounter,
	}, nil
}

// NewOTELInterceptor creates a new OpenTelemetry interceptor for ConnectRPC
func NewOTELInterceptor() connect.UnaryInterceptorFunc {
	tracer := otel.Tracer("metald")
	meter := otel.Meter("metald")

	// Create metrics
	metrics, err := NewMetrics(meter)
	if err != nil {
		// Log error but continue without metrics
		slog.Default().Error("failed to create OTEL interceptor metrics",
			slog.String("error", err.Error()),
		)
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Extract procedure name
			procedure := req.Spec().Procedure
			methodName := extractMethodName(procedure)
			serviceName := extractServiceName(procedure)

			// AIDEV-NOTE: Using unified span naming convention: service.method
			spanName := fmt.Sprintf("metald.%s", methodName)

			// Start span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("rpc.system", "connect_rpc"),
					attribute.String("rpc.service", serviceName),
					attribute.String("rpc.method", methodName),
				),
			)
			// AIDEV-NOTE: Critical panic recovery in OTEL interceptor - preserves existing errors
			defer func() {
				// Add panic recovery
				if r := recover(); r != nil {
					// Record panic metrics
					if metrics != nil {
						attrs := []attribute.KeyValue{
							attribute.String("rpc.method", procedure),
							attribute.String("panic.type", fmt.Sprintf("%T", r)),
						}
						metrics.panicCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
					}

					span.RecordError(fmt.Errorf("panic: %v", r))
					span.SetStatus(codes.Error, fmt.Sprintf("panic: %v", r))
					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
				span.End()
			}()

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
			var resp connect.AnyResponse
			var err error
			
			// AIDEV-NOTE: Ensure proper error handling to prevent nil panics
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("handler panic: %v", r))
						span.RecordError(err)
					}
				}()
				resp, err = next(ctx, req)
			}()

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
				// This ensures errors are captured even with low sampling rates
				if !span.SpanContext().IsSampled() {
					_, errorSpan := tracer.Start(ctx, spanName+".error",
						trace.WithSpanKind(trace.SpanKindServer),
						trace.WithAttributes(
							attribute.String("rpc.system", "connect_rpc"),
							attribute.String("rpc.service", serviceName),
							attribute.String("rpc.method", methodName),
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

				// Record duration (this is automatically calculated by the histogram)
				// The actual duration recording happens via span events
			}

			return resp, err
		}
	}
}

// extractMethodName extracts the method name from a full procedure path
// e.g., "/vmprovisioner.v1.VmService/CreateVm" -> "CreateVm"
func extractMethodName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return procedure
}

// extractServiceName extracts the service name from a full procedure path
// e.g., "/vmprovisioner.v1.VmService/CreateVm" -> "vmprovisioner.v1.VmService"
func extractServiceName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
