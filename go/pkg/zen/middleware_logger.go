package zen

import (
	"context"
	"log/slog"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// WithLogging returns middleware that logs information about each request.
// It captures the method, path, status code, and processing time.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithLogging(logger)},
//	    route,
//	)
func WithLogging(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			start := time.Now()
			nextErr := next(ctx, s)
			serviceLatency := time.Since(start)

			logger.DebugContext(ctx, "request",
				slog.String("method", s.r.Method),
				slog.String("path", s.r.URL.Path),
				slog.Int("status", int(s.responseStatus)),
				slog.Int64("latency", serviceLatency.Milliseconds()),
			)

			if nextErr != nil {
				logger.ErrorContext(ctx, nextErr.Error(),
					slog.String("method", s.r.Method),
					slog.String("path", s.r.URL.Path),
					slog.Int("status", int(s.responseStatus)))
			}

			return nextErr
		}
	}
}
