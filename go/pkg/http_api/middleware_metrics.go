package httpApi

import (
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

type metricsMiddleware[TRequest Redacter, TResponse Redacter] struct {
	next        Handler[TRequest, TResponse]
	eventBuffer EventBuffer
}

func NewMetricsMiddleware[TRequest Redacter, TResponse Redacter](eventBuffer EventBuffer) Middleware[TRequest, TResponse] {
	return func(next Handler[TRequest, TResponse]) Handler[TRequest, TResponse] {

		return &metricsMiddleware[TRequest, TResponse]{
			next:        next,
			eventBuffer: eventBuffer,
		}

	}
}

func (mw *metricsMiddleware[TRequest, TResponse]) Handle(sess *Session[TRequest, TResponse]) error {

	start := time.Now()

	nextErr := mw.next.Handle(sess)
	serviceLatency := time.Since(start)

	requestHeaders := []string{}
	for k, vv := range sess.r.Header {
		if k == "authorization" {
			requestHeaders = append(requestHeaders, "authorization: <REDACTED>")
		} else {
			requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
		}
	}

	responseHeaders := []string{}
	for k, vv := range sess.responseHeader {
		responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
	}
	req := sess.Request()
	body.Redact()
	requestBody, err :=

		mw.eventBuffer.BufferApiRequest(schema.ApiRequestV1{
			RequestID:       sess.RequestID(),
			Time:            start.UnixMilli(),
			Host:            sess.r.Host,
			Method:          sess.r.Method,
			Path:            sess.r.URL.Path,
			RequestHeaders:  requestHeaders,
			RequestBody:     string(),
			ResponseStatus:  summary.ResponseStatus,
			ResponseHeaders: responseHeaders,
			ResponseBody:    string(summary.ResponseBody),
			Error:           "",
			ServiceLatency:  serviceLatency.Milliseconds(),
		})
	return nextErr
}
