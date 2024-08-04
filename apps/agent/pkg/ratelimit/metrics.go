package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	activeRatelimits = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_active",
	}, []string{"identifier"})

	ratelimitsPassed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_passed",
	}, []string{"identifier"})
	ratelimitsRejected = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_rejected",
	}, []string{"identifier"})
)
