package zen

import (
	"context"
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

			logger.Debug("request",
				"method", s.r.Method,
				"path", s.r.URL.Path,
				"status", s.responseStatus,
				"latency", serviceLatency.Milliseconds(),
			)

			if nextErr != nil {
				logger.Error(nextErr.Error(),
					"method", s.r.Method,
					"path", s.r.URL.Path,
					"status", s.responseStatus)
			}
			return nextErr
		}
	}
}
