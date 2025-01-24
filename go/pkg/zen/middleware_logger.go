package zen

import (
	"log/slog"
	"time"

	"github.com/unkeyed/unkey/go/pkg/logging"
)

func WithLogging(logger logging.Logger) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(s *Session) error {
			start := time.Now()
			nextErr := next(s)
			serviceLatency := time.Since(start)

			logger.Info(s.Context(), "request",
				slog.String("method", s.r.Method),
				slog.String("path", s.r.URL.Path),
				slog.Int("status", s.responseStatus),
				slog.String("latency", serviceLatency.String()),
			)
			return nextErr
		}
	}
}
