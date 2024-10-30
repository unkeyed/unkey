package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/prometheus"
)

type responseWriterStatusInterceptor struct {
	w          http.ResponseWriter
	statusCode int
}

// Pass through
func (w *responseWriterStatusInterceptor) Header() http.Header {
	return w.w.Header()
}

// Pass through
func (w *responseWriterStatusInterceptor) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

// Capture and pass through
func (w *responseWriterStatusInterceptor) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.w.WriteHeader(statusCode)
}

func withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wi := &responseWriterStatusInterceptor{w: w}

		start := time.Now()
		next.ServeHTTP(wi, r)
		serviceLatency := time.Since(start)

		prometheus.HTTPRequests.With(map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
			"status": fmt.Sprintf("%d", wi.statusCode),
		}).Inc()

		prometheus.ServiceLatency.WithLabelValues(r.URL.Path).Observe(serviceLatency.Seconds())
	})
}
