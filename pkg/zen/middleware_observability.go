package zen

import (
	"context"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func WithObservability(m Metrics) Middleware {
	if m == nil {
		m = NoopMetrics{}
	}
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			ctx, span := tracing.Start(ctx, s.r.Pattern)
			span.SetAttributes(attribute.String("request_id", s.RequestID()))
			defer span.End()

			start := time.Now()

			err := next(ctx, s)
			if err != nil {
				tracing.RecordError(span, err)
			}

			serviceLatency := time.Since(start)
			status, _ := strconv.Atoi(strconv.Itoa(s.responseStatus))

			m.RecordHTTPRequest(s.r.Method, s.r.URL.Path, status, len(s.requestBody), serviceLatency.Seconds())

			return err
		}
	}
}
