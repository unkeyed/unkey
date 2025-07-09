package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// DebugInterceptor provides detailed debug logging for ConnectRPC calls
// AIDEV-NOTE: This interceptor logs detailed connection error information
// to help diagnose inter-service communication issues
func DebugInterceptor(logger *slog.Logger, serviceName string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			// Log request initiation at debug level
			logger.LogAttrs(ctx, slog.LevelDebug, fmt.Sprintf("%s rpc request initiated", serviceName),
				slog.String("procedure", procedure),
				slog.String("protocol", req.Spec().StreamType.String()),
			)

			// Execute the request
			resp, err := next(ctx, req)
			duration := time.Since(start)

			if err != nil { //nolint:nestif // Complex error logging logic requires nested conditions for different error types and connection troubleshooting
				// AIDEV-BUSINESS_RULE: Enhanced error logging for connection issues
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					// Connection error - log with full details
					attrs := []slog.Attr{
						slog.String("service", serviceName),
						slog.String("procedure", procedure),
						slog.Duration("duration", duration),
						slog.String("error", err.Error()),
						slog.String("code", connectErr.Code().String()),
						slog.String("message", connectErr.Message()),
					}

					// Add additional context for specific error codes
					switch connectErr.Code() {
					case connect.CodeUnavailable:
						attrs = append(attrs, slog.String("likely_cause", "service unreachable or down"))
					case connect.CodeDeadlineExceeded:
						attrs = append(attrs, slog.String("likely_cause", "request timeout - service may be overloaded"))
					case connect.CodePermissionDenied:
						attrs = append(attrs, slog.String("likely_cause", "authentication/authorization failure"))
					case connect.CodeUnauthenticated:
						attrs = append(attrs, slog.String("likely_cause", "missing or invalid credentials"))
					case connect.CodeCanceled:
						attrs = append(attrs, slog.String("likely_cause", "request was cancelled"))
					case connect.CodeUnknown:
						attrs = append(attrs, slog.String("likely_cause", "unknown server error"))
					case connect.CodeInvalidArgument:
						attrs = append(attrs, slog.String("likely_cause", "invalid request parameters"))
					case connect.CodeNotFound:
						attrs = append(attrs, slog.String("likely_cause", "requested resource not found"))
					case connect.CodeAlreadyExists:
						attrs = append(attrs, slog.String("likely_cause", "resource already exists"))
					case connect.CodeResourceExhausted:
						attrs = append(attrs, slog.String("likely_cause", "service resource limits exceeded"))
					case connect.CodeFailedPrecondition:
						attrs = append(attrs, slog.String("likely_cause", "operation precondition not met"))
					case connect.CodeAborted:
						attrs = append(attrs, slog.String("likely_cause", "operation was aborted"))
					case connect.CodeOutOfRange:
						attrs = append(attrs, slog.String("likely_cause", "operation out of valid range"))
					case connect.CodeUnimplemented:
						attrs = append(attrs, slog.String("likely_cause", "operation not implemented"))
					case connect.CodeInternal:
						attrs = append(attrs, slog.String("likely_cause", "internal server error"))
					case connect.CodeDataLoss:
						attrs = append(attrs, slog.String("likely_cause", "unrecoverable data loss"))
					}

					// Check if this is a connection refused error
					if strings.Contains(err.Error(), "connection refused") {
						attrs = append(attrs, slog.String("connection_status", "refused"))
						attrs = append(attrs, slog.String("troubleshooting", "check if target service is running and listening on the correct port"))
					}

					// Check for DNS resolution errors
					if strings.Contains(err.Error(), "no such host") {
						attrs = append(attrs, slog.String("connection_status", "dns_failure"))
						attrs = append(attrs, slog.String("troubleshooting", "check service endpoint configuration and DNS resolution"))
					}

					// Check for TLS errors
					if strings.Contains(err.Error(), "tls:") || strings.Contains(err.Error(), "x509:") {
						attrs = append(attrs, slog.String("connection_status", "tls_failure"))
						attrs = append(attrs, slog.String("troubleshooting", "check TLS certificates and configuration"))
					}

					logger.LogAttrs(ctx, slog.LevelError, fmt.Sprintf("%s connection error", serviceName), attrs...)
				} else {
					// Non-connection error
					logger.LogAttrs(ctx, slog.LevelError, fmt.Sprintf("%s rpc error", serviceName),
						slog.String("procedure", procedure),
						slog.Duration("duration", duration),
						slog.String("error", err.Error()),
						slog.String("error_type", fmt.Sprintf("%T", err)),
					)
				}
			} else {
				// Success - log at debug level
				logger.LogAttrs(ctx, slog.LevelDebug, fmt.Sprintf("%s rpc success", serviceName),
					slog.String("procedure", procedure),
					slog.Duration("duration", duration),
				)
			}

			return resp, err
		}
	}
}
