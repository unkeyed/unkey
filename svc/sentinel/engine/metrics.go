package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

var (
	// sentinelEngineEvaluationsTotal counts policy evaluations by type and result.
	//
	// Labels:
	//   - "policy_type": "keyauth" (future: "jwtauth", "ratelimit", etc.)
	//   - "result": "success", "denied", "error", "skipped"
	sentinelEngineEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_engine_evaluations_total",
			Help: "Total number of policy evaluations by type and result.",
		},
		[]string{"policy_type", "result"},
	)
)

// classifyKeyauthError maps a keyauth executor error to a metric result label.
// Auth-related failures (missing creds, invalid key, insufficient perms, rate limited)
// are "denied"; everything else is "error".
func classifyKeyauthError(err error) string {
	urn, ok := fault.GetCode(err)
	if !ok {
		return "error"
	}

	//nolint:exhaustive
	switch urn {
	case codes.Sentinel.Auth.MissingCredentials.URN(),
		codes.Sentinel.Auth.InvalidKey.URN(),
		codes.Sentinel.Auth.InsufficientPermissions.URN(),
		codes.Sentinel.Auth.RateLimited.URN():
		return "denied"
	default:
		return "error"
	}
}
