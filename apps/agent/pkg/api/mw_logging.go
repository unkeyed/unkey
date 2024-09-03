package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/api/ctxutil"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse"
	"github.com/unkeyed/unkey/apps/agent/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type responseWriterInterceptor struct {
	w          http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
}

// Pass through
func (w *responseWriterInterceptor) Header() http.Header {
	return w.w.Header()
}

// Capture and pass through
func (w *responseWriterInterceptor) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.w.Write(b)
}

// Capture and pass through
func (w *responseWriterInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.w.WriteHeader(statusCode)
}
func withLogging(next http.Handler, ch clickhouse.Bufferer, logger logging.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ctx := r.Context()
		wi := &responseWriterInterceptor{w: w, body: &bytes.Buffer{}}

		errorMessage := ""
		// r2 is a clone of r, so we can read the body twice
		r2 := r.Clone(ctx)
		defer r2.Body.Close()
		requestBody, err := io.ReadAll(r2.Body)
		if err != nil {
			logger.Error().Err(err).Msg("error reading r2 body")
			errorMessage = err.Error()
			requestBody = []byte("unable to read request body")
		}

		next.ServeHTTP(wi, r)
		serviceLatency := time.Since(start)

		logger.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Int("status", wi.statusCode).
			Str("latency", serviceLatency.String()).
			Msg("request")

		requestHeaders := []string{}
		for k, vv := range r.Header {
			if strings.ToLower(k) == "authorization" {
				vv = []string{"<REDACTED>"}
			}
			requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
		}

		responseHeaders := []string{}
		for k, vv := range wi.Header() {
			responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
		}

		ch.BufferApiRequest(schema.ApiRequestV1{
			RequestID:       ctxutil.GetRequestId(ctx),
			Time:            start.UnixMilli(),
			Host:            r.Host,
			Method:          r.Method,
			Path:            r.URL.Path,
			RequestHeaders:  requestHeaders,
			RequestBody:     string(requestBody),
			ResponseStatus:  wi.statusCode,
			ResponseHeaders: responseHeaders,
			ResponseBody:    wi.body.String(),
			Error:           errorMessage,
		})
	})
}
