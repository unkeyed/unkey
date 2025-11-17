package zen

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequest)
}

type redactionRule struct {
	regexp      *regexp.Regexp
	replacement []byte
}

var redactionRules = []redactionRule{
	// Redact "key" field values - matches JSON-style key fields with various whitespace combinations
	{
		regexp:      regexp.MustCompile(`"key"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"key": "[REDACTED]"`),
	},
	// Redact "plaintext" field values - matches JSON-style plaintext fields with various whitespace combinations
	{
		regexp:      regexp.MustCompile(`"plaintext"\s*:\s*"[^"\\]*(?:\\.[^"\\]*)*"`),
		replacement: []byte(`"plaintext": "[REDACTED]"`),
	},
}

func redact(in []byte) []byte {
	b := in

	for _, rule := range redactionRules {
		b = rule.regexp.ReplaceAll(b, rule.replacement)
	}

	return b
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
func WithMetrics(eventBuffer EventBuffer, info InstanceInfo) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			start := time.Now()
			nextErr := next(ctx, s)
			serviceLatency := time.Since(start)

			// Only log if we should log request to ClickHouse
			if s.ShouldLogRequestToClickHouse() {
				requestHeaders := []string{}
				for k, vv := range s.r.Header {
					if strings.ToLower(k) == "authorization" {
						requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, "[REDACTED]"))
					} else {
						requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
					}
				}

				responseHeaders := []string{}
				for k, vv := range s.w.Header() {
					responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
				}

				// https://docs.aws.amazon.com/elasticloadbalancing/latest/classic/x-forwarded-headers.html#x-forwarded-for
				ips := strings.Split(s.r.Header.Get("X-Forwarded-For"), ",")
				ipAddress := ""
				if len(ips) > 0 {
					ipAddress = ips[0]
				}

				eventBuffer.BufferApiRequest(schema.ApiRequest{
					WorkspaceID:     s.WorkspaceID,
					RequestID:       s.RequestID(),
					Time:            start.UnixMilli(),
					Host:            s.r.Host,
					Method:          s.r.Method,
					Path:            s.r.URL.Path,
					RequestHeaders:  requestHeaders,
					RequestBody:     string(redact(s.requestBody)),
					ResponseStatus:  int32(s.responseStatus),
					ResponseHeaders: responseHeaders,
					ResponseBody:    string(redact(s.responseBody)),
					Error:           fault.UserFacingMessage(nextErr),
					ServiceLatency:  serviceLatency.Milliseconds(),
					UserAgent:       s.r.Header.Get("User-Agent"),
					IpAddress:       ipAddress,
					Region:          info.Region,
					QueryString:     s.r.URL.Query().Encode(),
					QueryParams:     s.r.URL.Query(),
				})
			}
			return nextErr
		}
	}
}
