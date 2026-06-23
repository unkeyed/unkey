package handler

import (
	"bytes"
	"context"
	"io"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
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
		return h.ProxyService.ForwardToRegion(ctx, sess, decision.RemoteRegionAddress)
	}

	req := sess.Request()

	// The ClickHouse logging middleware seeds an empty tracking record
	// before this handler runs. Populate it now that the route resolved;
	// the retry loop and proxy callbacks fill in the per-attempt instance,
	// timing, and status as the request progresses.
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
	tracking.RequestID = sess.RequestID()
	tracking.DeploymentID = decision.DeploymentID
	tracking.WorkspaceID = decision.WorkspaceID
	tracking.EnvironmentID = decision.EnvironmentID
	tracking.ProjectID = decision.ProjectID

	// Tell the ClickHouse logging middleware which header and query-parameter
	// names carry an API key so it can redact them. KeyAuth supports custom
	// header and query-parameter key delivery; without this the raw key would
	// be persisted verbatim in the request log.
	tracking.SecretHeaders, tracking.SecretQueryParams = policies.SecretLocations(decision.Policies)

	// Evaluate policies before forwarding. The edge middleware has already
	// stripped any client-supplied X-Unkey-Principal header; if KeyAuth
	// produces a principal, we set it here for the upstream.
	if len(decision.Policies) > 0 && h.Engine != nil {
		result, evalErr := h.Engine.Evaluate(ctx, sess, req, decision.WorkspaceID, decision.Policies)
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

	// Capture the request body for ClickHouse via TeeReader. Bytes flow
	// to the upstream untouched while a copy accumulates in buf, capped
	// at MaxBodyCapture so a multi-GB upload cannot blow the heap. Works
	// for both streaming (gRPC, Connect) and unary requests.
	//
	// With the dial-failure retry loop, a failed first attempt does not
	// consume the body (the proxy never opened a TCP connection), so the
	// successful attempt drains it and the tee captures from that drain.
	// The captured body therefore always reflects what the *serving*
	// instance actually saw.
	if req.Body != nil {
		var buf bytes.Buffer
		req.Body = io.NopCloser(io.TeeReader(req.Body, &zen.LimitedWriter{W: &buf, N: zen.MaxBodyCapture}))
		defer func() {
			if buf.Len() > 0 {
				tracking.RequestBody = buf.Bytes()
			}
		}()
	}

	// Try each candidate instance in turn. We only move to the next
	// instance on dial-phase failures: the proxy never opened a TCP
	// connection, so the request body has not been read and replay is
	// safe. Any other error — mid-stream resets, response timeouts,
	// context cancellation — is returned to the client unchanged, since
	// the upstream may already have started processing the request and a
	// retry would risk double-execute on non-idempotent endpoints.
	//
	// 4xx / 5xx responses from the app are not errors at this layer; they
	// flow back through the proxy's ModifyResponse path and never reach
	// here.
	sawDialFailure := false
	var forwardErr error
	for _, instance := range decision.LocalInstances {
		tracking.InstanceID = instance.ID
		tracking.Address = instance.Address

		forwardErr = h.ProxyService.ForwardToInstance(ctx, sess, decision.UpstreamProtocol, instance)
		if forwardErr == nil {
			if sawDialFailure {
				localRequestRetriesTotal.WithLabelValues(retryOutcomeRecovered).Inc()
			}
			return nil
		}
		if !proxy.IsDialError(forwardErr) {
			return forwardErr
		}
		sawDialFailure = true
	}
	if sawDialFailure {
		localRequestRetriesTotal.WithLabelValues(retryOutcomeExhausted).Inc()
	}

	// Every local instance dial-failed (or there were none — shouldn't
	// happen since the router would have returned a remote decision in
	// that case, but treat it the same way). If the router gave us a
	// peer-region standby, fall through to it. The peer redoes its own
	// routing and retry. Without a standby, surface the last dial error.
	if decision.RemoteRegionAddress != "" {
		regionFallbacksTotal.WithLabelValues(decision.RemoteRegionAddress).Inc()
		return h.ProxyService.ForwardToRegion(ctx, sess, decision.RemoteRegionAddress)
	}

	// forwardErr is nil only when the loop never ran, i.e. LocalInstances
	// was empty *and* RemoteRegionAddress was empty. That's an invariant
	// violation: the router should have returned a remote decision instead
	// of a local one with no candidates. Without this guard, returning the
	// zero-value nil here surfaces to the client as a silent empty 200 —
	// fail closed with an explicit 503 so the bug is visible.
	if forwardErr == nil {
		return fault.New("local decision with no instances",
			fault.Code(codes.Frontline.Routing.NoRunningInstances.URN()),
			fault.Internal("router returned DestinationLocalInstance with empty LocalInstances"),
			fault.Public("Service temporarily unavailable"),
		)
	}
	return forwardErr
}
