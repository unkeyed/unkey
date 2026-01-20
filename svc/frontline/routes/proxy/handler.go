package handler

import (
	"context"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Handler struct {
	Logger        logging.Logger
	Region        string
	RouterService router.Service
	ProxyService  proxy.Service
	Clock         clock.Clock
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

// Handle routes incoming requests to either:
// 1. Local sentinel (if healthy sentinel in current region) - forwards with X-Unkey-Deployment-Id
// 2. Remote region (if no local sentinel) - forwards to nearest region
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	start := h.Clock.Now()
	ctx = proxy.WithRequestStartTime(ctx, start)
	ctx = h.ProxyService.InitTrace(ctx, sess)
	hostname := proxy.ExtractHostname(sess.Request().Host)

	route, sentinels, err := h.RouterService.LookupByHostname(ctx, hostname)
	if err != nil {
		return err
	}

	// Find Local sentinel or nearest NLB
	decision, err := h.RouterService.SelectSentinel(ctx, route, sentinels)
	if err != nil {
		return err
	}

	// We obviously prefer a local sentinel if available
	if decision.LocalSentinel != nil {
		return h.ProxyService.ForwardToSentinel(ctx, sess, decision.LocalSentinel, decision.DeploymentID)
	}

	return h.ProxyService.ForwardToRegion(ctx, sess, decision.NearestNLBRegion)
}
