package handler

import (
	"bytes"
	"context"
	"io"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

type Handler struct {
	RouterService router.Service
	ProxyService  proxy.Service
	Engine        policies.Evaluator
	Clock         clock.Clock
}

func (h *Handler) Method() string {
	return zen.CATCHALL
}

func (h *Handler) Path() string {
	return "/{path...}"
}

func (h *Handler) Handle(ctx context.Context, sess *zen.Session) error {
	startTime := h.Clock.Now()
	ctx = proxy.WithRequestStartTime(ctx, startTime)

	hostname := proxy.ExtractHostname(sess.Request().Host)

	decision, err := h.RouterService.Route(ctx, hostname)
	if err != nil {
		return err
	}

	if decision.Destination != router.DestinationLocalInstance {
		return h.ProxyService.Forward(ctx, sess, decision)
	}

	req := sess.Request()

	// The ClickHouse logging middleware seeds an empty tracking record
	// before this handler runs. Populate it now that the route resolved;
	// the engine + proxy callbacks fill in timing, status, and bodies as
	// the request progresses.
	tracking, ok := proxy.RequestTrackingFromContext(ctx)
	if !ok {
		// Defensive: register.go always wires the ClickHouse logging
		// middleware before this handler, so this branch is unreachable
		// in production. Allocate one so the engine + proxy don't panic
		// if someone reorders middleware.
		//nolint:exhaustruct
		tracking = &proxy.RequestTracking{StartTime: startTime}
		ctx = proxy.WithRequestTracking(ctx, tracking)
	}
	tracking.RequestID = uid.New("req")
	tracking.DeploymentID = decision.DeploymentID
	tracking.WorkspaceID = decision.WorkspaceID
	tracking.EnvironmentID = decision.EnvironmentID
	tracking.ProjectID = decision.ProjectID
	tracking.InstanceID = decision.Instance.ID
	tracking.Address = decision.Instance.Address

	// Evaluate policies before forwarding. The edge middleware has already
	// stripped any client-supplied X-Unkey-Principal header; if KeyAuth
	// produces a principal, we set it here for the upstream.
	if len(decision.Policies) > 0 && h.Engine != nil {
		result, evalErr := h.Engine.Evaluate(ctx, sess, req, decision.Policies)
		if evalErr != nil {
			return evalErr
		}
		if result.Principal != nil {
			principalJSON, serErr := result.Principal.Marshal()
			if serErr != nil {
				logger.Error("failed to serialize principal", "error", serErr)
			} else {
				req.Header.Set(policies.PrincipalHeader, principalJSON)
			}
		}
	}

	// Capture the request body for ClickHouse via TeeReader. Bytes flow to
	// the upstream untouched while a copy accumulates in buf, capped at
	// MaxBodyCapture so a multi-GB upload cannot blow the heap. Works for
	// both streaming (gRPC, Connect) and unary requests; the reverse proxy
	// drains the body exactly once on its way to the upstream.
	if req.Body != nil {
		var buf bytes.Buffer
		req.Body = io.NopCloser(io.TeeReader(req.Body, &zen.LimitedWriter{W: &buf, N: zen.MaxBodyCapture}))
		defer func() {
			if buf.Len() > 0 {
				tracking.RequestBody = buf.Bytes()
			}
		}()
	}

	return h.ProxyService.Forward(ctx, sess, decision)
}
