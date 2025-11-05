package middleware

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"go.opentelemetry.io/otel/attribute"
)

// WithTracing creates a middleware that adds tracing to requests
func WithTracing(logger logging.Logger) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			ctx, span := tracing.Start(ctx, s.Request().Pattern)
			span.SetAttributes(
				attribute.String("request_id", s.RequestID()),
				attribute.String("service", "ingress"),
				attribute.String("host", s.Request().Host),
				attribute.String("path", s.Request().URL.Path),
			)
			defer span.End()

			err := next(ctx, s)
			if err != nil {
				tracing.RecordError(span, err)
			}

			return err
		}
	}
}
