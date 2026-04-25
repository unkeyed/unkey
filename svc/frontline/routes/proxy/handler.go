package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/edgeredirect"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Handler struct {
	RouterService router.Service
	ProxyService  proxy.Service
	EdgeRedirect  edgeredirect.Evaluator
	Clock         clock.Clock
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	ctx = proxy.WithRequestStartTime(ctx, h.Clock.Now())
	hostname := proxy.ExtractHostname(sess.Request().Host)

	decision, err := h.RouterService.Route(ctx, hostname)
	if err != nil {
		return err
	}

	// Per-FQDN edge-redirect rules short-circuit the proxy when matched.
	// The common case (no rules, e.g. auto-generated preview URLs) is a
	// single len() check.
	if len(decision.Redirects) > 0 {
		if res := h.EdgeRedirect.Evaluate(sess.Request(), decision.Redirects); res != nil {
			sess.ResponseWriter().Header().Set("Location", res.Location)
			edgeredirect.EdgeRedirectsTotal.WithLabelValues(res.RuleKind).Inc()
			return sess.Send(res.Status, nil)
		}
	}

	return h.ProxyService.Forward(ctx, sess, decision)
}
