package zen

import (
	"context"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/redaction"
)

// ApiRequestBuffer abstracts the method used by WithMetrics to buffer API request events.
// *batch.BatchProcessor[schema.ApiRequest] satisfies this interface.
type ApiRequestBuffer interface {
	Buffer(schema.ApiRequest)
}

var skipHeaders = map[string]bool{
	"x-forwarded-proto": true,
	"x-forwarded-port":  true,
	"x-forwarded-for":   true,
	"x-amzn-trace-id":   true,
}

// credentialHeaders carry a credential in their value, so only the name is
// logged. Cookie is here because portal routes authenticate with a session
// cookie: the token is redacted from the one response that mints it, and without
// this it would be written in the clear by every request that then uses it.
var credentialHeaders = map[string]bool{
	"authorization":       true,
	"cookie":              true,
	"set-cookie":          true,
	"proxy-authorization": true,
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
// Request and response bodies are buffered verbatim, so redactor decides what
// reaches storage. Pass the one built from the service's OpenAPI spec; a nil
// redactor stores bodies unchanged and is only appropriate for services whose
// payloads carry no credentials.
//
// The buffered row's body strings alias the session's body slices when there was
// nothing to redact, which keeps the common request allocation-free. That holds
// only because those slices are replaced per request rather than refilled in
// place; see the invariant on [Session.requestBody].
//
// Example:
//
//	server.RegisterRoute(
//	    []zen.Middleware{zen.WithMetrics(eventBuffer, info, redactor)},
//	    route,
//	)
func WithMetrics(apiRequestBuffer ApiRequestBuffer, info InstanceInfo, redactor *redaction.Redactor) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			start := time.Now()
			nextErr := next(ctx, s)
			serviceLatency := time.Since(start)

			// Only log if we should log request to ClickHouse
			if s.ShouldLogRequestToClickHouse() {
				requestHeaders := make([]string, 0, len(s.r.Header))
				for k, vv := range s.r.Header {
					lk := strings.ToLower(k)
					if skipHeaders[lk] {
						continue
					}

					if credentialHeaders[lk] {
						requestHeaders = append(requestHeaders, formatHeader(k, "[REDACTED]"))
					} else {
						requestHeaders = append(requestHeaders, formatHeader(k, strings.Join(vv, ",")))
					}
				}

				responseHeaders := make([]string, 0, len(s.w.Header()))
				for k, vv := range s.w.Header() {
					if credentialHeaders[strings.ToLower(k)] {
						responseHeaders = append(responseHeaders, formatHeader(k, "[REDACTED]"))

						continue
					}
					responseHeaders = append(responseHeaders, formatHeader(k, strings.Join(vv, ",")))
				}

				workspaceID := ""
				if principal, err := s.GetPrincipal(); err == nil {
					workspaceID = principal.WorkspaceID
				}

				apiRequestBuffer.Buffer(schema.ApiRequest{
					WorkspaceID:     workspaceID,
					RequestID:       s.RequestID(),
					Time:            start.UnixMilli(),
					Host:            s.r.Host,
					Method:          s.r.Method,
					Path:            s.r.URL.Path,
					QueryString:     s.r.URL.RawQuery,
					QueryParams:     s.r.URL.Query(),
					RequestHeaders:  requestHeaders,
					RequestBody:     redactor.RedactString(s.requestBody),
					ResponseStatus:  int32(s.responseStatus),
					ResponseHeaders: responseHeaders,
					ResponseBody:    redactor.RedactString(s.responseBody),
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
