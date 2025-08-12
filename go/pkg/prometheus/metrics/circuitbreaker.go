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
		Namespace:   "unkey",
		Subsystem:   "circuitbreaker",
		Name:        "requests_total",
		Help:        "Tracks the number of requests made to the circuitbreaker by state.",
		ConstLabels: constLabels,
	}, []string{"service", "action"})

	// CircuitBreakerErrorsTotal tracks the total number of circuit breaker errors,
	// labeled by service and action. Use this counter to monitor circuit breaker error rates.
	//
	// Example usage:
	//   metrics.CircuitBreakerErrorsTotal.WithLabelValues("database", "timeout").Inc()
	CircuitBreakerErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "circuitbreaker",
		Name:        "errors_total",
		Help:        "Total number of circuit breaker errors by service and action.",
		ConstLabels: constLabels,
	}, []string{"service", "action"})
)
