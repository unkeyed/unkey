package zen

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/zen/metrics"
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

			// "method", "path", "status".
			//
			// path is the routed pattern (e.g. "/v1/keys.verifyKey" or
			// "/{path...}" for a catch-all), NOT s.r.URL.Path. URL.Path
			// is user-controlled and unbounded — on services like
			// frontline that proxy arbitrary customer URLs, using it
			// here would cause a cardinality explosion in Prometheus.
			//
			// s.r.Pattern is registered as "METHOD /path" (see
			// Server.RegisterRoute); strip the method prefix so the path
			// label stays orthogonal to the separate method label and
			// matches the bare-path values used before.
			pattern := s.r.Pattern
			if _, rest, found := strings.Cut(pattern, " "); found {
				pattern = rest
			}
			labelValues := []string{s.r.Method, pattern, strconv.Itoa(s.responseStatus)}

			metrics.HTTPRequestBodySize.WithLabelValues(labelValues...).Observe(float64(len(s.requestBody)))
			metrics.HTTPRequestTotal.WithLabelValues(labelValues...).Inc()
			metrics.HTTPRequestLatency.WithLabelValues(labelValues...).Observe(serviceLatency.Seconds())

			return err
		}
	}
}
