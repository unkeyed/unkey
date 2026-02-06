package zen

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/logger"
)

// WithLogging returns middleware that logs information about each request.
// It captures the method, path, status code, and processing time.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithLogging()},
//	    route,
//	)
func WithLogging() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {

			ctx, event := logger.StartWideEvent(ctx,
				fmt.Sprintf("%s %s", s.r.Method, s.r.URL.Path),
				slog.Group("http_request",
					slog.String("method", s.r.Method),
					slog.String("path", s.r.URL.Path),
					slog.String("request_id", s.RequestID()),
					slog.String("host", s.r.URL.Host),
					slog.String("user_agent", s.r.UserAgent()),
					slog.String("ip_address", s.Location()),
				),
			)

			defer event.End()

			nextErr := next(ctx, s)

			event.SetError(nextErr)

			event.Set(
				slog.Group("http_response",
					slog.Int("status_code", s.StatusCode()),
				),
			)
			return nextErr
		}
	}
}
