// Package gcra connects the rate limit handlers to the global limiter in
// pkg/ratelimit, which uses GCRA (Generic Cell Rate Algorithm). Each
// namespace selects the limiter with its global_ratelimit_mode:
//
//	off     the regional limiter decides.
//	shadow  the regional limiter decides, and the global limiter decides the
//	        same call in the background for metrics only.
//	on      the global limiter decides.
package gcra

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/metrics"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	globalratelimit "github.com/unkeyed/unkey/pkg/ratelimit"
)

const (
	// shadowDecisionsMax bounds the shadow decisions that run at once, so
	// shadow mode cannot pile up goroutines while the global limiter is
	// slow. A call beyond it is counted as skipped.
	shadowDecisionsMax = 1_024

	// shadowTimeout bounds one shadow decision, which may wait for a read.
	shadowTimeout = time.Second
)

// Limiter decides calls with the global limiter and runs shadow decisions.
type Limiter struct {
	limiter     globalratelimit.Limiter
	shadowSlots chan struct{}
}

// New returns a Limiter on `limiter`. It is safe to call concurrently.
func New(limiter globalratelimit.Limiter) *Limiter {
	return &Limiter{limiter: limiter, shadowSlots: make(chan struct{}, shadowDecisionsMax)}
}

// ModeOf returns the mode of a call over namespaces with `modes`: on if every
// namespace is on, shadow if every namespace is shadow or on, and off
// otherwise. A call is all or nothing, so one limiter decides all of it, and
// a shadow decision only compares calls that the global limiter could decide
// whole. It is safe to call concurrently.
func ModeOf(modes ...db.RatelimitNamespacesGlobalRatelimitMode) db.RatelimitNamespacesGlobalRatelimitMode {
	mode := db.RatelimitNamespacesGlobalRatelimitModeOn
	for _, m := range modes {
		switch m {
		case db.RatelimitNamespacesGlobalRatelimitModeOn:
		case db.RatelimitNamespacesGlobalRatelimitModeShadow:
			mode = db.RatelimitNamespacesGlobalRatelimitModeShadow
		case db.RatelimitNamespacesGlobalRatelimitModeOff:
			return db.RatelimitNamespacesGlobalRatelimitModeOff
		default:
			return db.RatelimitNamespacesGlobalRatelimitModeOff
		}
	}
	return mode
}

// Allow decides `reqs` all or nothing with the global limiter and counts each
// decision in metrics.RatelimitDecision. It returns a validation error if two
// requests have the same fingerprint, which overrides can cause after the
// request values passed the uniqueness check. It is safe to call
// concurrently.
func (l *Limiter) Allow(ctx context.Context, reqs []ratelimit.RatelimitRequest) ([]ratelimit.RatelimitResponse, error) {
	requests, err := toGlobalRequests(reqs)
	if err != nil {
		return nil, err
	}
	results, err := l.limiter.AllowMany(ctx, requests)
	if err != nil {
		return nil, err
	}
	responses := make([]ratelimit.RatelimitResponse, len(results))
	for i, result := range results {
		metrics.RatelimitDecision.WithLabelValues(reqs[i].WorkspaceID, "gcra", outcomeOf(result.Allowed)).Inc()
		responses[i] = ratelimit.RatelimitResponse{
			Limit:     reqs[i].Limit,
			Remaining: result.Remaining,
			Reset:     result.ResetAt,
			Success:   result.Allowed,
			Current:   0,
		}
	}
	return responses, nil
}

// Shadow decides `reqs` with the global limiter in the background and counts
// the result next to `regionalPassed`, the decision the caller got, in
// metrics.RatelimitShadowDecisions. The global limiter spends units as if it
// had decided, so its state follows the traffic it would see when the
// namespace is on. Nothing it returns reaches the caller. `op` is "limit" or
// "multi_limit". It is safe to call concurrently.
func (l *Limiter) Shadow(ctx context.Context, op string, reqs []ratelimit.RatelimitRequest, regionalPassed bool) {
	workspaceID := reqs[0].WorkspaceID
	regional := outcomeOf(regionalPassed)
	select {
	case l.shadowSlots <- struct{}{}:
	default:
		metrics.RatelimitShadowDecisions.WithLabelValues(workspaceID, op, regional, "skipped").Inc()
		return
	}
	go func() {
		defer func() { <-l.shadowSlots }()
		shadowCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shadowTimeout)
		defer cancel()
		start := time.Now()
		outcome := "error"
		requests, err := toGlobalRequests(reqs)
		if err == nil {
			var results []globalratelimit.Result
			results, err = l.limiter.AllowMany(shadowCtx, requests)
			if err == nil {
				outcome = outcomeOf(allAllowed(results))
			}
		}
		metrics.RatelimitDecisionDuration.WithLabelValues(op, "gcra_shadow").Observe(time.Since(start).Seconds())
		metrics.RatelimitShadowDecisions.WithLabelValues(workspaceID, op, regional, outcome).Inc()
	}()
}

// toGlobalRequests builds the requests for the global limiter. It is safe to
// call concurrently.
func toGlobalRequests(reqs []ratelimit.RatelimitRequest) ([]globalratelimit.Request, error) {
	requests := make([]globalratelimit.Request, len(reqs))
	fingerprints := make(map[[32]byte]struct{}, len(reqs))
	for i, req := range reqs {
		fingerprint := namespace.GlobalFingerprint(req.WorkspaceID, req.Namespace, req.Identifier, req.Duration)
		if _, duplicate := fingerprints[fingerprint]; duplicate {
			return nil, fault.New("duplicate rate limit check after overrides",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Each check must be unique by namespace, identifier and duration after overrides are applied."),
			)
		}
		fingerprints[fingerprint] = struct{}{}
		requests[i] = globalratelimit.Request{
			Fingerprint: fingerprint,
			Rate:        globalratelimit.Rate{Limit: req.Limit, Duration: req.Duration},
			Units:       req.Cost,
		}
	}
	return requests, nil
}

// allAllowed reports whether every result is allowed. It is safe to call
// concurrently.
func allAllowed(results []globalratelimit.Result) bool {
	for _, result := range results {
		if !result.Allowed {
			return false
		}
	}
	return true
}

// outcomeOf returns the metric label of a decision. It is safe to call
// concurrently.
func outcomeOf(allowed bool) string {
	if allowed {
		return "passed"
	}
	return "denied"
}
