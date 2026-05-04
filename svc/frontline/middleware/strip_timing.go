package middleware

import (
	"context"

	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
)

func WithTimingDisabled() zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			return next(timing.WithDisabled(ctx), s)
		}
	}
}
