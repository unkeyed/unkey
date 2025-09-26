package zen

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// WithTracing returns middleware that adds OpenTelemetry tracing to each request.
// It creates a span for the entire request lifecycle and propagates context.
//
// If an error occurs during handling, it will be recorded in the span.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithTracing()},
//	    route,
//	)
func WithTracing() Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			ctx, span := tracing.Start(ctx, s.r.Pattern)
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
