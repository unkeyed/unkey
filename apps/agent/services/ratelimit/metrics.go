package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ratelimitAccuracy = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimit_accuracy",
	}, []string{"correct"})

	activeRatelimits = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_active",
	})

	ratelimitsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_total",
	}, []string{"passed"})

	// forceSync is a counter that increments every time the agent is forced to
	// sync with the origin ratelimit service because it doesn't have enough data
	forceSync = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "force_sync",
	})
)
