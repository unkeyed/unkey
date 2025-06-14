package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

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

// NewOTELInterceptor creates a ConnectRPC interceptor that adds OpenTelemetry tracing
func NewOTELInterceptor() connect.UnaryInterceptorFunc {
	tracer := otel.Tracer("builderd/rpc")
	meter := otel.Meter("builderd/rpc")

	// Create metrics
	metrics, err := NewMetrics(meter)
	if err != nil {
		// Log error but continue without metrics
		slog.Default().Error("failed to create OTEL interceptor metrics",
			slog.String("error", err.Error()),
		)
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
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
			// AIDEV-NOTE: Critical panic recovery in OTEL interceptor - preserves existing errors
			defer func() {
				// Add panic recovery
				if r := recover(); r != nil {
					// Get stack trace
					stack := debug.Stack()
					
					// Log detailed panic information
					slog.Default().Error("panic in OTEL interceptor",
						slog.String("procedure", procedure),
						slog.Any("panic", r),
						slog.String("panic_type", fmt.Sprintf("%T", r)),
						slog.String("stack_trace", string(stack)),
					)

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
			resp, err = next(ctx, req)

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

// NewLoggingInterceptor creates a ConnectRPC interceptor that logs all RPC calls
func NewLoggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// Add panic recovery
			defer func() {
				if r := recover(); r != nil {
					// Get stack trace
					stack := debug.Stack()
					
					logger.Error("panic in logging interceptor",
						slog.String("procedure", req.Spec().Procedure),
						slog.Any("panic", r),
						slog.String("panic_type", fmt.Sprintf("%T", r)),
						slog.String("stack_trace", string(stack)),
					)
					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
			}()

			start := time.Now()

			// Extract trace ID if available
			var traceID string
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}

			// Log request
			logger.LogAttrs(ctx, slog.LevelInfo, "rpc request",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("protocol", req.Peer().Protocol),
				slog.String("trace_id", traceID),
				slog.String("user_agent", req.Header().Get("User-Agent")),
			)

			// Execute request
			resp, err = next(ctx, req)

			// Log response
			duration := time.Since(start)
			if err != nil {
				// Determine log level based on error type
				logLevel := slog.LevelError
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					switch connectErr.Code() {
					case connect.CodeNotFound, connect.CodeAlreadyExists:
						logLevel = slog.LevelWarn
					case connect.CodeInvalidArgument, connect.CodeFailedPrecondition:
						logLevel = slog.LevelWarn
					case connect.CodeUnauthenticated, connect.CodePermissionDenied:
						logLevel = slog.LevelWarn
					}
				}

				logger.LogAttrs(ctx, logLevel, "rpc error",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
					slog.String("trace_id", traceID),
				)
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "rpc success",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
					slog.String("trace_id", traceID),
				)
			}

			return resp, err
		}
	}
}

// NewTenantAuthInterceptor creates a ConnectRPC interceptor for tenant authentication
func NewTenantAuthInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// Add panic recovery
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic in tenant auth interceptor",
						slog.String("procedure", req.Spec().Procedure),
						slog.Any("panic", r),
					)
					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
			}()
			// Extract tenant information from headers
			tenantID := req.Header().Get("X-Tenant-ID")
			customerID := req.Header().Get("X-Customer-ID")
			authToken := req.Header().Get("Authorization")

			// Add tenant context to the request context
			ctx = withTenantContext(ctx, TenantAuthContext{
				TenantID:   tenantID,
				CustomerID: customerID,
				AuthToken:  authToken,
			})

			// Add tenant info to span if tracing is enabled
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				span.SetAttributes(
					attribute.String("tenant.id", tenantID),
					attribute.String("tenant.customer_id", customerID),
				)
			}

			// For now, we'll implement basic validation
			// In production, this would validate the auth token and tenant permissions
			if tenantID == "" && req.Spec().Procedure != "/builder.v1.BuilderService/GetBuildStats" {
				logger.LogAttrs(ctx, slog.LevelWarn, "missing tenant ID",
					slog.String("procedure", req.Spec().Procedure),
				)
				return nil, connect.NewError(connect.CodeUnauthenticated,
					fmt.Errorf("tenant ID is required"))
			}

			logger.LogAttrs(ctx, slog.LevelDebug, "tenant authenticated",
				slog.String("tenant_id", tenantID),
				slog.String("customer_id", customerID),
				slog.String("procedure", req.Spec().Procedure),
			)

			return next(ctx, req)
		}
	}
}

// TenantAuthContext holds tenant authentication information
type TenantAuthContext struct {
	TenantID   string
	CustomerID string
	AuthToken  string
}

// contextKey is a private type for context keys
type contextKey string

const tenantContextKey contextKey = "tenant_auth"

// withTenantContext adds tenant auth context to the context
func withTenantContext(ctx context.Context, auth TenantAuthContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, auth)
}

// TenantFromContext extracts tenant auth context from the context
func TenantFromContext(ctx context.Context) (TenantAuthContext, bool) {
	auth, ok := ctx.Value(tenantContextKey).(TenantAuthContext)
	return auth, ok
}
