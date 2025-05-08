package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// CircuitBreakerRequests tracks the number of requests made to the circuit breaker.
	// This counter helps monitor the usage and performance of the circuit breaker.
	//
	// Example usage:
	//   metrics.CircuitBreakerRequests.WithLabelValues("my_circuit_breaker", "open").Inc()
	CircuitBreakerRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem:   "circuitbreaker",
		Name:        "requests_total",
		Help:        "Tracks the number of requests made to the circuitbreaker by state.",
		ConstLabels: constLabels,
	}, []string{"name", "state"})
)
