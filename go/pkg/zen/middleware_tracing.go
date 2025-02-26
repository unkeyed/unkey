package zen

import "github.com/unkeyed/unkey/go/pkg/otel/tracing"

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
