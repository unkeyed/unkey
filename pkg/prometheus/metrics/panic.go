package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the panic recovery package.
type Metrics struct {
	// PanicsTotal tracks panics recovered by HTTP handler middleware.
	// Labels:
	//   - "caller": The function or handler that panicked
	//   - "path": The HTTP request path that triggered the panic
	PanicsTotal *prometheus.CounterVec
}

// NewMetrics creates and registers all panic metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	f := promauto.With(reg)

	return &Metrics{
		PanicsTotal: f.NewCounterVec(prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "internal",
			Name:      "panics_total",
			Help:      "Total number of panics recovered in HTTP handlers.",
		}, []string{"caller", "path"}),
	}
}
