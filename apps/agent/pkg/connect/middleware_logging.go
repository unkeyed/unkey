package connect

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type loggingMiddleware struct {
	handler http.Handler
	logger  logging.Logger
}

func newLoggingMiddleware(handler http.Handler, logger logging.Logger) http.Handler {
	return &loggingMiddleware{handler, logger}
}

func (h *loggingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	h.handler.ServeHTTP(w, r)
	h.logger.Info().Str("method", r.Method).Str("path", r.URL.Path).Int64("serviceLatency", time.Since(start).Milliseconds()).Msg("request")

}
