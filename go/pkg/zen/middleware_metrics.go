package zen

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}

// WithMetrics returns middleware that collects metrics about each request,
// including request counts, latencies, and status codes.
//
// The metrics are buffered and periodically sent to an event buffer.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithMetrics(eventBuffer)},
//	    route,
//	)
func WithMetrics(eventBuffer EventBuffer) Middleware {
	redactions := map[*regexp.Regexp]string{
		regexp.MustCompile(`"key":\s*"[a-zA-Z0-9_]+"`):       `"key": "[REDACTED]"`,
		regexp.MustCompile(`"plaintext":\s*"[a-zA-Z0-9_]+"`): `"plaintext": "[REDACTED]"`,
	}

	redact := func(in []byte) []byte {
		b := in
		for r, replacement := range redactions {
			b = r.ReplaceAll(b, []byte(replacement))
		}
		return b
	}

	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			start := time.Now()
			nextErr := next(ctx, s)
			serviceLatency := time.Since(start)

			requestHeaders := []string{}
			for k, vv := range s.r.Header {
				if k == "authorization" {
					requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, "[REDACTED]"))
				} else {
					requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
				}
			}

			responseHeaders := []string{}
			for k, vv := range s.w.Header() {
				responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
			}

			metrics.Http.Requests.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(
				attribute.String("host", s.r.Host),
				attribute.String("method", s.r.Method),
				attribute.String("path", s.r.URL.Path),
				attribute.Int("status", s.responseStatus),
			)))

			eventBuffer.BufferApiRequest(schema.ApiRequestV1{
				WorkspaceID:     s.workspaceID,
				RequestID:       s.requestID,
				Time:            start.UnixMilli(),
				Host:            s.r.Host,
				Method:          s.r.Method,
				Path:            s.r.URL.Path,
				RequestHeaders:  requestHeaders,
				RequestBody:     string(redact(s.requestBody)),
				ResponseStatus:  s.responseStatus,
				ResponseHeaders: responseHeaders,
				ResponseBody:    string(redact(s.responseBody)),
				Error:           "",
				ServiceLatency:  serviceLatency.Milliseconds(),
			})
			return nextErr
		}
	}
}
