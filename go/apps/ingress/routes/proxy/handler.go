package handler

import (
	"context"

	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	"github.com/unkeyed/unkey/go/apps/ingress/services/router"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
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
// 1. Local gateway (if healthy gateway in current region) - forwards with X-Unkey-Deployment-Id
// 2. Remote NLB (if no local gateway) - forwards to nearest region's NLB
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	startTime := h.Clock.Now()
	hostname := sess.Request().Host

	// Lookup route and gateways by hostname
	route, gateways, err := h.RouterService.LookupByHostname(ctx, hostname)
	if err != nil {
		return err // Error already has proper fault wrapping
	}

	// Select best gateway (local or find nearest NLB)
	decision, err := h.RouterService.SelectGateway(route, gateways)
	if err != nil {
		return err
	}

	// Route to local gateway if available
	if decision.LocalGateway != nil {
		return h.ProxyService.ForwardToGateway(ctx, sess, decision.LocalGateway, decision.DeploymentID, startTime)
	}

	// Route to remote NLB
	return h.ProxyService.ForwardToNLB(ctx, sess, decision.NearestNLBRegion, startTime)
}
