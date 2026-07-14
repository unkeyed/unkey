package zen

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/logger"
)

// LoggingOption configures the WithLogging middleware.
type LoggingOption func(*loggingConfig)

type loggingConfig struct {
	skipPrefixes []string
}

// SkipPaths configures path prefixes that should not be logged.
// Any request whose path starts with one of these prefixes will
// skip logging entirely.
//
// Example:
//
//	zen.WithLogging(zen.SkipPaths("/_unkey/internal/", "/health/"))
func SkipPaths(prefixes ...string) LoggingOption {
	return func(cfg *loggingConfig) {
		cfg.skipPrefixes = append(cfg.skipPrefixes, prefixes...)
	}
}

// WithLogging returns middleware that logs failed request information.
// It captures the method, path, status code, and processing time.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithLogging(zen.SkipPaths("/_unkey/internal/", "/health/"))},
//	    route,
//	)
func WithLogging(opts ...LoggingOption) Middleware {
	cfg := &loggingConfig{
		skipPrefixes: nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			for _, prefix := range cfg.skipPrefixes {
				if strings.HasPrefix(s.r.URL.Path, prefix) {
					return next(ctx, s)
				}
			}

			ctx, event := logger.StartWideEvent(ctx,
				fmt.Sprintf("%s %s", s.r.Method, s.r.URL.Path),
			)

			nextErr := next(ctx, s)

			event.SetError(nextErr)
			event.Set(
				slog.Group("http",
					slog.String("method", s.r.Method),
					slog.String("path", s.r.URL.Path),
					slog.String("request_id", s.RequestID()),
					slog.String("host", s.r.Host),
					slog.String("user_agent", s.r.UserAgent()),
					slog.String("ip_address", s.Location()),
					slog.Int("status_code", s.StatusCode()),
				),
			)

			if nextErr != nil || s.StatusCode() >= http.StatusBadRequest {
				event.End()
			}

			return nextErr
		}
	}
}
