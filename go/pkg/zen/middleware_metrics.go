package zen

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

type metricsMiddleware struct {
	next        Handler
	eventBuffer EventBuffer

	redactions map[*regexp.Regexp]string
}

func NewMetricsMiddleware[TRequest Redacter, TResponse Redacter](eventBuffer EventBuffer) Middleware {
	return func(next Handler) Handler {

		return &metricsMiddleware{
			next:        next,
			eventBuffer: eventBuffer,
			redactions: map[*regexp.Regexp]string{
				regexp.MustCompile(`"key":\s*"[a-zA-Z0-9_]+"`):       `"key": "[REDACTED]"`,
				regexp.MustCompile(`"plaintext":\s*"[a-zA-Z0-9_]+"`): `"plaintext": "[REDACTED]"`,
			},
		}

	}
}

// I'm not super happy with this as it seems inefficient compared to creating one
// regular expression with all replacements, but for now it shall be fine.
//
// If it becomes a performance issue, it'll be an easy fix.
func (mw *metricsMiddleware) redact(in []byte) []byte {
	b := in
	for r, replacement := range mw.redactions {
		b = r.ReplaceAll(b, []byte(replacement))
	}
	return b
}

func (mw *metricsMiddleware) Handle(sess *Session) error {

	start := time.Now()
	nextErr := mw.next.Handle(sess)
	serviceLatency := time.Since(start)
	requestHeaders := []string{}
	for k, vv := range sess.r.Header {
		if k == "authorization" {
			requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, "[REDACTED]"))
		} else {
			requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
		}
	}

	responseHeaders := []string{}
	for k, vv := range sess.w.Header() {
		responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
	}

	mw.eventBuffer.BufferApiRequest(schema.ApiRequestV1{
		RequestID:       sess.RequestID(),
		Time:            start.UnixMilli(),
		Host:            sess.r.Host,
		Method:          sess.r.Method,
		Path:            sess.r.URL.Path,
		RequestHeaders:  requestHeaders,
		RequestBody:     string(mw.redact(sess.requestBody)),
		ResponseStatus:  sess.responseStatus,
		ResponseHeaders: responseHeaders,
		ResponseBody:    string(mw.redact(sess.responseBody)),
		Error:           "",
		ServiceLatency:  serviceLatency.Milliseconds(),
	})
	return nextErr
}
