package zen

import (
	"context"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"go.opentelemetry.io/otel/attribute"
)

// WithObservability returns middleware that adds OpenTelemetry metrics and tracing to each request.
// It creates a span for the entire request lifecycle and propagates context.
//
// If an error occurs during handling, it will be recorded in the span.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithObservability()},
//	    route,
//	)
func WithObservability() Middleware {
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

			// "method", "path", "status"
			labelValues := []string{s.r.Method, s.r.URL.Path, strconv.Itoa(s.responseStatus)}

			metrics.HTTPRequestBodySize.WithLabelValues(labelValues...).Observe(float64(len(s.requestBody)))
			metrics.HTTPRequestTotal.WithLabelValues(labelValues...).Inc()
			metrics.HTTPRequestLatency.WithLabelValues(labelValues...).Observe(serviceLatency.Seconds())

			return err
		}
	}
}
