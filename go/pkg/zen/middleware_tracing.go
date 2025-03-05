package zen

import "github.com/unkeyed/unkey/go/pkg/otel/tracing"

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
		return func(s *Session) error {
			ctx, span := tracing.Start(s.Context(), s.r.Pattern)
			defer span.End()

			s.ctx = ctx

			err := next(s)
			if err != nil {
				tracing.RecordError(span, err)
			}
			return err
		}
	}
}
