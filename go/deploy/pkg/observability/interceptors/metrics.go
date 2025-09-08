package interceptors

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the OTEL metrics for the interceptor.
type Metrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	activeRequests  metric.Int64UpDownCounter
	panicCounter    metric.Int64Counter
}

// NewMetrics creates new metrics using the provided meter.
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

// NewMetricsInterceptor creates a ConnectRPC interceptor that collects OpenTelemetry metrics
// and provides distributed tracing for all RPC calls.
//
// AIDEV-NOTE: This interceptor provides consistent metrics collection across all Unkey services
func NewMetricsInterceptor(opts ...Option) connect.UnaryInterceptorFunc {
	options := applyOptions(opts)

	// Create metrics if meter is provided
	var metrics *Metrics
	if options.Meter != nil {
		m, err := NewMetrics(options.Meter)
		if err != nil {
			// Log error but continue without metrics
			if options.Logger != nil {
				options.Logger.Error("failed to create metrics",
					slog.String("service", options.ServiceName),
					slog.String("error", err.Error()),
				)
			} else {
				slog.Default().Error("failed to create metrics",
					slog.String("service", options.ServiceName),
					slog.String("error", err.Error()),
				)
			}
		} else {
			metrics = m
		}
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			start := time.Now()

			// Extract procedure info using shared utilities
			procedure := req.Spec().Procedure
			methodName := tracing.ExtractMethodName(procedure)
			serviceName := tracing.ExtractServiceName(procedure)

			// AIDEV-NOTE: Using unified span naming convention: service.method
			spanName := tracing.FormatSpanName(options.ServiceName, methodName)

			// Start span
			tracer := otel.Tracer(options.ServiceName)
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("rpc.system", "connect_rpc"),
					attribute.String("rpc.service", serviceName),
					attribute.String("rpc.method", methodName),
				),
			)

			defer func() {
				if r := recover(); r != nil {
					// Log panic with optional stack trace
					if options.Logger != nil {
						attrs := []any{
							slog.String("service", options.ServiceName),
							slog.String("procedure", procedure),
							slog.Any("panic", r),
							slog.String("panic_type", fmt.Sprintf("%T", r)),
						}
						if options.EnablePanicStackTrace {
							attrs = append(attrs, slog.String("stack_trace", string(debug.Stack())))
						}
						options.Logger.Error("panic in metrics interceptor", attrs...)
					}

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

			// Track active requests if enabled
			if metrics != nil && options.EnableActiveRequestsMetric {
				attrs := []attribute.KeyValue{
					attribute.String("rpc.method", procedure),
				}
				metrics.activeRequests.Add(ctx, 1, metric.WithAttributes(attrs...))
				defer metrics.activeRequests.Add(ctx, -1, metric.WithAttributes(attrs...))
			}

			// Call the handler with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("metrics handler panic: %v", r))
						span.RecordError(err)
					}
				}()
				resp, err = next(ctx, req)
			}()

			// Calculate duration
			duration := time.Since(start)

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

				// Error resampling: create a new span that's always sampled
				// This ensures errors are captured even with low sampling rates
				if options.EnableErrorResampling && !span.SpanContext().IsSampled() {
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

				// Record duration if enabled
				if options.EnableRequestDurationMetric {
					metrics.requestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
				}
			}

			return resp, err
		}
	}
}
