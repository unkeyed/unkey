package connect

import (
	"net/http"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type tracingMiddleware struct {
	tracer  tracing.Tracer
	handler http.Handler
}

func newTracingMiddleware(handler http.Handler, tracer tracing.Tracer) http.Handler {
	return &tracingMiddleware{handler: handler, tracer: tracer}
}

func (h *tracingMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.tracer.Start(r.Context(), "request")
	defer span.End()

	h.handler.ServeHTTP(w, r.WithContext(ctx))

}
