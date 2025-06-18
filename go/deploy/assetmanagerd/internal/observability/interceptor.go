package observability

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Metrics holds the OTEL metrics for the service
type Metrics struct {
	requestCounter  metric.Int64Counter
	requestDuration metric.Float64Histogram
	panicCounter    metric.Int64Counter
}

var (
	globalMetrics *Metrics
	tracer        = GetTracer("assetmanagerd")
)

func init() {
	meter := GetMeter("assetmanagerd")

	requestCounter, _ := meter.Int64Counter("rpc_server_requests_total",
		metric.WithDescription("Total number of RPC requests"),
		metric.WithUnit("1"),
	)

	requestDuration, _ := meter.Float64Histogram("rpc_server_request_duration_seconds",
		metric.WithDescription("RPC request duration in seconds"),
		metric.WithUnit("s"),
	)

	panicCounter, _ := meter.Int64Counter("rpc_server_panics_total",
		metric.WithDescription("Total number of RPC server panics"),
		metric.WithUnit("1"),
	)

	globalMetrics = &Metrics{
		requestCounter:  requestCounter,
		requestDuration: requestDuration,
		panicCounter:    panicCounter,
	}
}

// NewMetricsInterceptor creates a new OTEL metrics interceptor
func NewMetricsInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// AIDEV-NOTE: Panic recovery in OTEL interceptor with proper error preservation
			// This ensures panics are tracked in metrics and don't crash the server
			defer func() {
				if r := recover(); r != nil {
					if globalMetrics != nil && globalMetrics.panicCounter != nil {
						globalMetrics.panicCounter.Add(ctx, 1,
							metric.WithAttributes(
								attribute.String("rpc.method", req.Spec().Procedure),
								attribute.String("panic.type", fmt.Sprintf("%T", r)),
							),
						)
					}

					// Get or create span for proper error recording
					span := trace.SpanFromContext(ctx)
					if span.IsRecording() {
						span.RecordError(fmt.Errorf("panic: %v", r))
						span.SetStatus(codes.Error, "panic recovered")
					}

					// Convert panic to error if the handler didn't already return one
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", r))
					}
				}
			}()

			start := time.Now()
			procedure := req.Spec().Procedure
			methodName := extractMethodName(procedure)
			serviceName := extractServiceName(procedure)

			// AIDEV-NOTE: Using unified span naming convention: service.method
			spanName := fmt.Sprintf("assetmanagerd.%s", methodName)

			// Create span
			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.RPCService(serviceName),
					semconv.RPCMethod(methodName),
					semconv.RPCSystemKey.String("connectrpc"),
				),
			)
			defer span.End()

			// Call the handler
			resp, err = next(ctx, req)
			duration := time.Since(start).Seconds()

			// Determine status
			var status string
			var code string
			if err != nil {
				code = connect.CodeOf(err).String()
				status = "error"
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			} else {
				code = "OK"
				status = "success"
				span.SetStatus(codes.Ok, "")
			}

			// Record metrics
			attrs := []attribute.KeyValue{
				attribute.String("rpc.method", procedure),
				attribute.String("rpc.status", status),
				attribute.String("rpc.code", code),
			}

			if globalMetrics != nil {
				if globalMetrics.requestCounter != nil {
					globalMetrics.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
				}
				if globalMetrics.requestDuration != nil {
					globalMetrics.requestDuration.Record(ctx, duration, metric.WithAttributes(attrs...))
				}
			}

			return resp, err
		}
	}
}

// NewLoggingInterceptor creates a new logging interceptor
func NewLoggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// AIDEV-NOTE: Panic recovery in logging interceptor for defense in depth
			// Preserves existing errors and logs panic details for debugging
			defer func() {
				if r := recover(); r != nil {
					logger.LogAttrs(ctx, slog.LevelError, "panic in RPC handler",
						slog.String("procedure", req.Spec().Procedure),
						slog.Any("panic", r),
						slog.String("panic_type", fmt.Sprintf("%T", r)),
					)

					// Convert panic to error if the handler didn't already return one
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("panic: %v", r))
					}
				}
			}()

			start := time.Now()
			procedure := req.Spec().Procedure

			// Log request
			logger.LogAttrs(ctx, slog.LevelInfo, "RPC request started",
				slog.String("procedure", procedure),
				slog.String("peer", req.Peer().Addr),
			)

			// Call the handler
			resp, err = next(ctx, req)
			duration := time.Since(start)

			// Log response
			if err != nil {
				logger.LogAttrs(ctx, slog.LevelError, "RPC request failed",
					slog.String("procedure", procedure),
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
					slog.String("code", connect.CodeOf(err).String()),
				)
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "RPC request completed",
					slog.String("procedure", procedure),
					slog.Duration("duration", duration),
				)
			}

			return resp, err
		}
	}
}

// extractMethodName extracts the method name from a full procedure path
func extractMethodName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return procedure
}

// extractServiceName extracts the service name from a full procedure path
func extractServiceName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}
