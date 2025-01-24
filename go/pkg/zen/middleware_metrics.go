package zen

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

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
		return func(s *Session) error {
			start := time.Now()
			nextErr := next(s)
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

			eventBuffer.BufferApiRequest(schema.ApiRequestV1{
				RequestID:       s.RequestID(),
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
