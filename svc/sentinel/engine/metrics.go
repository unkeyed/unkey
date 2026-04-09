package engine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

var (
	sentinelEngineEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "engine_evaluations_total",
			Help:      "Total number of policy evaluations by type and result.",
		},
		[]string{"policy_type", "result"},
	)

	sentinelEngineEvaluationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "unkey",
			Subsystem: "sentinel",
			Name:      "engine_evaluation_duration_seconds",
			Help:      "Duration of individual policy evaluations in seconds.",
			Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"policy_type"},
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
	case codes.Sentinel.Auth.RateLimited.URN():
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
	case codes.Sentinel.Auth.MissingCredentials.URN(),
		codes.Sentinel.Auth.InvalidKey.URN(),
		codes.Sentinel.Auth.InsufficientPermissions.URN(),
		codes.Sentinel.Auth.RateLimited.URN():
		return "denied"
	default:
		return "error"
	}
}
