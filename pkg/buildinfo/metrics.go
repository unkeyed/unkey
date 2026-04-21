package buildinfo

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
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

// RegisterBuildInfoMetrics emits the unkey_build_info gauge for the given
// service name, tagged with the link-time build identity (Version, Revision,
// Go runtime version, BuildTime).
//
// Call once during service startup, after lazy.SetRegistry has installed the
// prometheus registry that should receive the metric.
func RegisterBuildInfoMetrics(service string) {
	buildInfo.WithLabelValues(service, Version, Revision, runtime.Version(), BuildTime).Set(1)
}
