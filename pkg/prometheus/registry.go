package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// NewServiceRegistry returns a registry with the Go runtime, process, and host
// system collectors, and installs it as the target of [lazy] metrics. Expose it
// with [Handler].
//
// Call it once at service startup, before any lazy metric is used. Because
// [lazy.SetRegistry] keeps the first registry it receives, a later call still
// returns a registry with the runtime collectors but lazy metrics remain on
// the first one.
func NewServiceRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(collectors.NewGoCollector())
	reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	reg.MustRegister(NewSystemMetricsCollector())
	lazy.SetRegistry(reg)
	return reg
}

// Handler serves the metrics gathered from reg in the Prometheus exposition
// format.
func Handler(reg *prometheus.Registry) http.Handler {
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
