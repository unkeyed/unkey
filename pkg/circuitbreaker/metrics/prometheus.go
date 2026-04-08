package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the circuit breaker package.
type Metrics struct {
	// CircuitBreakerRequests tracks the number of requests made to the circuit breaker.
	CircuitBreakerRequests *prometheus.CounterVec

	// CircuitBreakerErrorsTotal tracks the total number of circuit breaker errors,
	// labeled by service and action.
	CircuitBreakerErrorsTotal *prometheus.CounterVec
}

// NewMetrics creates and registers all circuit breaker metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		CircuitBreakerRequests: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "circuitbreaker",
			Name:      "requests_total",
			Help:      "Tracks the number of requests made to the circuitbreaker by state.",
		}, []string{"service", "action"}),

		CircuitBreakerErrorsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "circuitbreaker",
			Name:      "errors_total",
			Help:      "Total number of circuit breaker errors by service and action.",
		}, []string{"service", "action"}),
	}
}

// NoopMetrics returns a Metrics instance registered to a discarded registry.
func NoopMetrics() *Metrics {
	return NewMetrics(prometheus.NewRegistry())
}
