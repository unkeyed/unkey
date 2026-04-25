// Package handler is the catchall on the plain-HTTP listener. It hands
// the request to the edge-redirect engine and writes the resulting 308.
// Runs without logging/observability middleware: every browser-typed
// http:// URL hits this path, so it must stay cheap. Volume is tracked via
// the unkey_frontline_https_redirects_total counter.
package handler

import (
	"context"
	"net/http"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/edgeredirect"
)

// Handler 308-redirects plain-HTTP requests via the edge-redirect engine.
// The Rules slice is built at registration time and never mutated.
type Handler struct {
	Engine edgeredirect.Evaluator
	Rules  []*edgeredirectv1.Rule
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

// Handle evaluates the engine and writes the redirect. Returns 404 if no
// rule matches, which is unreachable while a RequireHTTPS rule is configured
// but kept as a defensive fall-through.
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	res := h.Engine.Evaluate(sess.Request(), h.Rules)
	if res == nil {
		return sess.Send(http.StatusNotFound, nil)
	}
	sess.ResponseWriter().Header().Set("Location", res.Location)
	redirectsTotal.Inc()
	return sess.Send(res.Status, nil)
}
