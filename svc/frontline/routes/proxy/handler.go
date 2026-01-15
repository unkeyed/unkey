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
	hostname := proxy.ExtractHostname(sess.Request().Host)

	lookupStart := h.Clock.Now()
	route, sentinels, err := h.RouterService.LookupByHostname(ctx, hostname)
	lookupDuration := h.Clock.Now().Sub(lookupStart)

	h.Logger.Debug("frontline lookup complete",
		"hostname", hostname,
		"lookup_duration_ms", lookupDuration.Milliseconds(),
	)

	if err != nil {
		return err
	}

	// Find Local sentinel or nearest NLB
	selectStart := h.Clock.Now()
	decision, err := h.RouterService.SelectSentinel(route, sentinels)
	selectDuration := h.Clock.Now().Sub(selectStart)

	h.Logger.Debug("frontline select sentinel complete",
		"deployment_id", decision.DeploymentID,
		"has_local_sentinel", decision.LocalSentinel != nil,
		"select_duration_ms", selectDuration.Milliseconds(),
		"total_pre_forward_ms", h.Clock.Now().Sub(start).Milliseconds(),
	)

	if err != nil {
		return err
	}

	// We obviously prefer a local sentinel if available
	if decision.LocalSentinel != nil {
		return h.ProxyService.ForwardToSentinel(ctx, sess, decision.LocalSentinel, decision.DeploymentID)
	}

	return h.ProxyService.ForwardToRegion(ctx, sess, decision.NearestNLBRegion)
}
