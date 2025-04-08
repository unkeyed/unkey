package zen

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/prometheus/metrics"
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

			// "method", "path", "status"
			labelValues := []string{s.r.Method, s.r.URL.Path, strconv.Itoa(s.responseStatus)}

			metrics.HTTPRequestTotal.WithLabelValues(labelValues...).Inc()
			metrics.HTTPRequestLatency.WithLabelValues(labelValues...).Observe(serviceLatency.Seconds())

			// https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/x-forwarded-headers.html#x-forwarded-for
			ips := strings.Split(s.r.Header.Get("X-Forwarded-For"), ",")
			ipAddress := ""
			if len(ips) > 0 {
				ipAddress = ips[0]
			}

			if s.r.Header.Get("X-Unkey-Metrics") != "disabled" {
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
					Error:           fault.UserFacingMessage(nextErr),
					ServiceLatency:  serviceLatency.Milliseconds(),
					UserAgent:       s.r.Header.Get("User-Agent"),
					IpAddress:       ipAddress,
					Country:         "",
					City:            "",
					Colo:            "",
					Continent:       "",
				})
			}
			return nextErr
		}
	}
}
