package policies

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	engineEvaluationsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "engine_evaluations_total",
			Help:      "Total number of policy evaluations by type and result.",
		},
		[]string{"policy_type", "result"},
	)

	engineEvaluationDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "engine_evaluation_duration_seconds",
			Help:      "Duration of individual policy evaluations in seconds.",
			Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"policy_type"},
	)

	// firewallMatchesTotal counts individual Firewall policy matches,
	// labeled by the specific policy id and the action applied. The MVP only
	// emits action="deny", but the label is kept so added actions (log,
	// challenge, ...) don't require a breaking metric change.
	firewallMatchesTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "frontline",
			Name:      "firewall_matches_total",
			Help:      "Total number of Firewall policy matches by policy and action.",
		},
		[]string{"policy_id", "action"},
	)
)

// classifyRatelimitError maps a ratelimit executor error to a metric result label.
func classifyRatelimitError(err error) string {
	urn, ok := fault.GetCode(err)
	if !ok {
		return "error"
	}

	//nolint:exhaustive
	switch urn {
	case codes.Frontline.Auth.RateLimited.URN():
		return "denied"
	default:
		return "error"
	}
}

// classifyKeyauthError maps a keyauth executor error to a metric result label.
func classifyKeyauthError(err error) string {
	urn, ok := fault.GetCode(err)
	if !ok {
		return "error"
	}

	//nolint:exhaustive
	switch urn {
	case codes.Frontline.Auth.MissingCredentials.URN(),
		codes.Frontline.Auth.InvalidKey.URN(),
		codes.Frontline.Auth.InsufficientPermissions.URN(),
		codes.Frontline.Auth.RateLimited.URN():
		return "denied"
	default:
		return "error"
	}
}

// classifyFirewallError maps a firewall executor error to a metric result label.
// The executor only returns an error for DENY; anything else is an unexpected
// failure.
func classifyFirewallError(err error) string {
	urn, ok := fault.GetCode(err)
	if !ok {
		return "error"
	}
	if urn == codes.Frontline.Firewall.Denied.URN() {
		return "denied"
	}
	return "error"
}
