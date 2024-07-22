package connect

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
)

type metricsMiddleware struct {
	handler http.Handler
	metrics metrics.Metrics
}

func newMetricsMiddleware(handler http.Handler, metrics metrics.Metrics) http.Handler {
	return &metricsMiddleware{handler, metrics}
}

func (h *metricsMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.handler.ServeHTTP(w, r)
	servicelatency := time.Since(start).Milliseconds()

	h.metrics.Record(metrics.HttpRequest{
		Method:         r.Method,
		Path:           r.URL.Path,
		ServiceLatency: servicelatency,
		UserAgent:      r.UserAgent(),
		RemoteAddr:     r.RemoteAddr,
		SourceIP:       r.Header.Get("Fly-Client-IP"),
	})

}
