// Package buildinfometrics exposes build metadata as a Prometheus metric.
package buildinfometrics

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// buildInfo is a constant-1 gauge whose labels carry the build identity of
// the running service. Scrapers join against it to attribute metrics to a
// specific build (e.g. for canary comparisons or regression hunts).
//
// Labels: service, version, revision (full git SHA), goversion, build_time (RFC3339 UTC).
var buildInfo = lazy.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "unkey",
	Name:      "build_info",
	Help:      "Build metadata for the running service. Always 1; identity is in the labels.",
}, []string{"service", "version", "revision", "goversion", "build_time"})

// Register emits the unkey_build_info gauge for the given service name. Call
// it after lazy.SetRegistry installs the service's Prometheus registry.
func Register(service string) {
	buildInfo.WithLabelValues(
		service,
		buildinfo.Version,
		buildinfo.Revision,
		runtime.Version(),
		buildinfo.BuildTime,
	).Set(1)
}
