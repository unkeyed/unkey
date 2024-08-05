package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	activeRatelimits = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_active",
	})

	ratelimitsPassed = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_passed",
	})
	ratelimitsRejected = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_rejected",
	})
)
