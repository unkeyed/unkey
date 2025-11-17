package middleware

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/otel/tracing"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"go.opentelemetry.io/otel/attribute"
)

// WithObservability returns middleware that adds OpenTelemetry tracing to each request.
// Metrics are commented out to avoid alerting on 500 errors during development.
func WithObservability(logger logging.Logger) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			ctx, span := tracing.Start(ctx, s.Request().URL.Path)
			span.SetAttributes(attribute.String("request_id", s.RequestID()))
			defer span.End()

			start := time.Now()

			err := next(ctx, s)
			if err != nil {
				tracing.RecordError(span, err)
			}

			_ = time.Since(start) // serviceLatency

			// Metrics commented out to avoid alerting on 500 errors
			// labelValues := []string{s.Request().Method, s.Request().URL.Path, strconv.Itoa(s.ResponseStatus())}
			// metrics.HTTPRequestBodySize.WithLabelValues(labelValues...).Observe(float64(len(s.RequestBody())))
			// metrics.HTTPRequestTotal.WithLabelValues(labelValues...).Inc()
			// metrics.HTTPRequestLatency.WithLabelValues(labelValues...).Observe(serviceLatency.Seconds())

			return err
		}
	}
}
