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

	ratelimitsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "ratelimit",
		Name:      "ratelimits_total",
	}, []string{"passed"})
)
