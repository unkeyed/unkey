package router

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the router service package.
type Metrics struct {
	// DeploymentLookupTotal counts deployment lookup outcomes.
	//
	// Labels:
	//   outcome: "not_found", "error"
	DeploymentLookupTotal *prometheus.CounterVec

	// InstanceSelectionTotal counts instance selection outcomes.
	//
	// Labels:
	//   outcome: "success", "no_instances", "no_running_instances", "error"
	InstanceSelectionTotal *prometheus.CounterVec

	// RoutingDuration tracks how long routing operations take.
	//
	// Labels:
	//   operation: "get_deployment", "select_instance"
	RoutingDuration *prometheus.HistogramVec
}

// NewMetrics creates and registers all router service metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		DeploymentLookupTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "routing_deployment_lookup_total",
				Help:      "Total number of deployment lookup attempts by outcome.",
			},
			[]string{"outcome"},
		),
		InstanceSelectionTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "routing_instance_selection_total",
				Help:      "Total number of instance selection attempts by outcome.",
			},
			[]string{"outcome"},
		),
		RoutingDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "routing_duration_seconds",
				Help:      "Duration of routing operations in seconds.",
				Buckets:   []float64{0.0005, 0.001, 0.002, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
			},
			[]string{"operation"},
		),
	}

	reg.MustRegister(m.DeploymentLookupTotal)
	reg.MustRegister(m.InstanceSelectionTotal)
	reg.MustRegister(m.RoutingDuration)

	return m
}
