package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// CircuitBreakerRequests tracks the number of requests made to the circuit breaker.
	// This counter helps monitor the usage and performance of the circuit breaker.
	//
	// Example usage:
	//   metrics.CircuitBreakerRequests.WithLabelValues("my_circuit_breaker", "open").Inc()
	CircuitBreakerRequests = lazy.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "circuitbreaker",
		Name:      "requests_total",
		Help:      "Tracks the number of requests made to the circuitbreaker by state.",
	}, []string{"service", "action"})
)
