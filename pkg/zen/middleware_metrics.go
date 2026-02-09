package zen

import (
	"context"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
)

type EventBuffer interface {
	BufferApiRequest(schema.ApiRequest)
}

var skipHeaders = map[string]bool{
	"x-forwarded-proto": true,
	"x-forwarded-port":  true,
	"x-forwarded-for":   true,
	"x-amzn-trace-id":   true,
}

func formatHeader(key, value string) string {
	var b strings.Builder
	b.Grow(len(key) + 2 + len(value))
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	return b.String()
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
					lk := strings.ToLower(k)
					if skipHeaders[lk] {
						continue
					}

					if lk == "authorization" {
						requestHeaders = append(requestHeaders, formatHeader(k, "[REDACTED]"))
					} else {
						requestHeaders = append(requestHeaders, formatHeader(k, strings.Join(vv, ",")))
					}
				}

				responseHeaders := []string{}
				for k, vv := range s.w.Header() {
					responseHeaders = append(responseHeaders, formatHeader(k, strings.Join(vv, ",")))
				}

				eventBuffer.BufferApiRequest(schema.ApiRequest{
					WorkspaceID:     s.WorkspaceID,
					RequestID:       s.RequestID(),
					Time:            start.UnixMilli(),
					Host:            s.r.Host,
					Method:          s.r.Method,
					Path:            s.r.URL.Path,
					QueryString:     s.r.URL.RawQuery,
					QueryParams:     s.r.URL.Query(),
					RequestHeaders:  requestHeaders,
					RequestBody:     string(redact(s.requestBody)),
					ResponseStatus:  int32(s.responseStatus),
					ResponseHeaders: responseHeaders,
					ResponseBody:    string(redact(s.responseBody)),
					Error:           s.InternalError(),
					ServiceLatency:  serviceLatency.Milliseconds(),
					UserAgent:       s.r.Header.Get("User-Agent"),
					IpAddress:       s.Location(),
					Region:          info.Region,
				})
			}

			return nextErr
		}
	}
}
