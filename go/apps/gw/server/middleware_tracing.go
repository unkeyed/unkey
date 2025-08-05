package server

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// WithTracing returns middleware that adds OpenTelemetry tracing to each request.
// It creates a span for the entire request lifecycle and propagates context.
func WithTracing() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			ctx, span := tracing.Start(ctx, s.r.URL.Path)
			span.SetAttributes(attribute.String("request_id", s.RequestID()))
			defer span.End()

			err := next(ctx, s)
			if err != nil {
				tracing.RecordError(span, err)
			}

			return err
		}
	}
}
