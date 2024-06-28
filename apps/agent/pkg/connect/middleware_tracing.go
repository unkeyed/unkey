package connect

import (
	"net/http"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type tracingMiddleware struct {
	handler http.Handler
}

func newTracingMiddleware(handler http.Handler) http.Handler {
	return &tracingMiddleware{handler: handler}
}

func (h *tracingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracing.Start(r.Context(), "request")
	defer span.End()

	h.handler.ServeHTTP(w, r.WithContext(ctx))

}
