package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Handler struct {
	Logger            logging.Logger
	Region            string
	DeploymentService deployments.Service
	ProxyService      proxy.Service
	Clock             interface{ Now() time.Time }
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

// Handle routes incoming requests to either:
// 1. Local gateway (if deployment is in current region) - forwards to HTTP
// 2. Remote ingress (if deployment is elsewhere) - forwards to HTTPS
func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	startTime := h.Clock.Now()
	hostname := sess.Request().Host

	// Check for too many hops to prevent infinite routing loops
	if hopCountStr := sess.Request().Header.Get(proxy.HeaderIngressHops); hopCountStr != "" {
		if hops, err := strconv.Atoi(hopCountStr); err == nil && hops > h.ProxyService.GetMaxHops() {
			h.Logger.Error("too many ingress hops - rejecting request",
				"hops", hops,
				"hostname", hostname,
				"requestID", sess.RequestID(),
			)
			return fault.New("too many ingress hops",
				fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
				fault.Internal(fmt.Sprintf("request exceeded maximum hop count: %d", hops)),
				fault.Public("Request routing limit exceeded"),
			)
		}
	}

	// Lookup deployment by hostname
	deployment, found, err := h.DeploymentService.LookupByHostname(ctx, hostname)
	if err != nil {
		h.Logger.Error("failed to lookup deployment", "hostname", hostname, "error", err)
		return fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
			fault.Internal("deployment lookup failed"),
			fault.Public("Unable to process request"),
		)
	}

	// Deployment not found
	if !found {
		return fault.New("deployment not found",
			fault.Code(codes.Ingress.Routing.ConfigNotFound.URN()),
			fault.Internal(fmt.Sprintf("no deployment found for hostname: %s", hostname)),
			fault.Public("Domain not configured"),
		)
	}

	// Determine target based on region
	// if deployment.Region == h.Region {
	// 	// Local gateway - deployment is in this region
	// 	return h.ProxyService.ForwardToLocal(ctx, sess, deployment, startTime)
	// }

	// Remote ingress - route DIRECTLY to the deployment's region
	// No intermediate hops needed - we always route straight to where the deployment lives
	// return h.ProxyService.ForwardToRemote(ctx, sess, deployment.Region, deployment, startTime)
	return h.ProxyService.ForwardToRemote(ctx, sess, "", deployment, startTime)
}
