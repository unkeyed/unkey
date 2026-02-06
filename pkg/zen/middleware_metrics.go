package zen

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/zen/validation"
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

// buildRawRequestHeaders formats request headers with basic infra filtering.
// Used as fallback when WithValidation hasn't run (e.g. chproxy routes).
func buildRawRequestHeaders(r *http.Request) []string {
	headers := make([]string, 0, len(r.Header))
	for k, vv := range r.Header {
		lk := strings.ToLower(k)
		if skipHeaders[lk] {
			continue
		}
		headers = append(headers, validation.FormatHeader(k, strings.Join(vv, ",")))
	}
	return headers
}

// buildRawResponseHeaders formats response headers. Used as fallback when
// WithValidation hasn't run.
func buildRawResponseHeaders(w http.ResponseWriter) []string {
	headers := make([]string, 0, len(w.Header()))
	for k, vv := range w.Header() {
		headers = append(headers, validation.FormatHeader(k, strings.Join(vv, ",")))
	}
	return headers
}

// WithMetrics returns middleware that collects metrics about each request,
// including request counts, latencies, and status codes.
//
// The metrics are buffered and periodically sent to an event buffer.
//
// When WithValidation runs in the middleware chain, sanitized data is read
// from the Session. Otherwise, raw data is used with basic fallback formatting.
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithMetrics(eventBuffer, info)},
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
				eventBuffer.BufferApiRequest(schema.ApiRequest{
					WorkspaceID:     s.WorkspaceID,
					RequestID:       s.RequestID(),
					Time:            start.UnixMilli(),
					Host:            s.r.Host,
					Method:          s.r.Method,
					Path:            s.r.URL.Path,
					QueryString:     s.r.URL.RawQuery,
					QueryParams:     s.r.URL.Query(),
					RequestHeaders:  s.LoggableRequestHeaders(),
					RequestBody:     s.LoggableRequestBody(),
					ResponseStatus:  int32(s.responseStatus),
					ResponseHeaders: s.LoggableResponseHeaders(),
					ResponseBody:    s.LoggableResponseBody(),
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
