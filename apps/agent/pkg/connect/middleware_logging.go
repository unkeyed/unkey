package connect

import (
	"net/http"

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
	h.logger.Info().Str("method", r.Method).Str("path", r.URL.Path).Msg("request")
	h.handler.ServeHTTP(w, r)

}
