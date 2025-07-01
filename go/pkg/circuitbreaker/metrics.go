package circuitbreaker

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requests = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "agent",
		Subsystem: "circuitbreaker",
		Name:      "requests_total",
		Help:      "Total number of requests processed by circuit breaker",
	}, []string{"name", "state"})
)
