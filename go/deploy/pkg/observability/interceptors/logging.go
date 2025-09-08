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
	"go.opentelemetry.io/otel/trace"
)

// NewLoggingInterceptor creates a ConnectRPC interceptor that provides structured logging
// for all RPC calls, including request/response details, timing, and error information.
func NewLoggingInterceptor(opts ...Option) connect.UnaryInterceptorFunc {
	options := applyOptions(opts)

	// Use default logger if none provided
	logger := options.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// Preserves existing errors and logs panic details for debugging
			defer func() {
				if r := recover(); r != nil {
					attrs := []any{
						slog.String("service", options.ServiceName),
						slog.String("procedure", req.Spec().Procedure),
						slog.Any("panic", r),
						slog.String("panic_type", fmt.Sprintf("%T", r)),
					}
					if options.EnablePanicStackTrace {
						attrs = append(attrs, slog.String("stack_trace", string(debug.Stack())))
					}
					logger.Error("panic in logging interceptor", attrs...)

					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
			}()

			start := time.Now()
			procedure := req.Spec().Procedure
			methodName := tracing.ExtractMethodName(procedure)

			// Extract trace ID if available
			var traceID string
			if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}

			// Build request attributes
			requestAttrs := []slog.Attr{
				slog.String("service", options.ServiceName),
				slog.String("procedure", procedure),
				slog.String("method", methodName),
				slog.String("protocol", req.Peer().Protocol),
				slog.String("trace_id", traceID),
			}

			// Log request
			logger.LogAttrs(ctx, slog.LevelInfo, "rpc request started", requestAttrs...)

			// Execute request with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("logging handler panic: %v", r))
					}
				}()
				resp, err = next(ctx, req)
			}()

			// Calculate duration
			duration := time.Since(start)

			// Build response attributes
			responseAttrs := []slog.Attr{
				slog.String("service", options.ServiceName),
				slog.String("procedure", procedure),
				slog.Duration("duration", duration),
				slog.String("trace_id", traceID),
			}

			// Log response based on error status
			if err != nil {
				// Determine log level based on error type
				logLevel := slog.LevelError
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					responseAttrs = append(responseAttrs,
						slog.String("error", err.Error()),
						slog.String("code", connectErr.Code().String()),
					)

					// Use warning level for client-side errors
					switch connectErr.Code() {
					case connect.CodeNotFound,
						connect.CodeAlreadyExists,
						connect.CodeInvalidArgument,
						connect.CodeFailedPrecondition,
						connect.CodeUnauthenticated,
						connect.CodePermissionDenied,
						connect.CodeCanceled,
						connect.CodeDeadlineExceeded,
						connect.CodeResourceExhausted,
						connect.CodeAborted,
						connect.CodeOutOfRange:
						logLevel = slog.LevelWarn
					case connect.CodeUnknown,
						connect.CodeUnimplemented,
						connect.CodeInternal,
						connect.CodeUnavailable,
						connect.CodeDataLoss:
						logLevel = slog.LevelError
					}
				} else {
					responseAttrs = append(responseAttrs,
						slog.String("error", err.Error()),
					)
				}

				logger.LogAttrs(ctx, logLevel, "rpc request failed", responseAttrs...)
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "rpc request completed", responseAttrs...)
			}

			return resp, err
		}
	}
}
