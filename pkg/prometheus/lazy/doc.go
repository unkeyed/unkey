// Package lazy provides lazy-registering Prometheus metric wrappers.
//
// Metrics are declared as package-level globals (like promauto) but don't
// register until first use. Each service calls [SetRegistry] at startup
// to control where metrics end up. Only metrics that are actually used by
// a service get registered to that service's registry.
//
// Usage:
//
//	// Declare globally — same feel as promauto:
//	var RequestsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
//	    Namespace: "unkey",
//	    Subsystem: "api",
//	    Name:      "requests_total",
//	}, []string{"method", "status"})
//
//	// In service startup (one line):
//	reg := prometheus.NewRegistry()
//	lazy.SetRegistry(reg)
//
//	// When code calls RequestsTotal.WithLabelValues("GET", "200").Inc(),
//	// it registers to the service's registry on first use. If the service
//	// never touches this metric, it never registers — zero pollution.
//
// Pre-create series you alert or graph on. A series that does not exist yet
// is missing, not zero: rate() and increase() ignore its first increment,
// error-ratio alerts evaluate to no data, and dashboards show gaps.
//
// For a metric without labels ([Counter], [Gauge], [Histogram]), call Register
// after SetRegistry to export it at 0. A metric with labels ([CounterVec],
// [GaugeVec], [HistogramVec]) exports one series per combination of label
// values, and none until a combination is used. Create each combination you
// know about with WithLabelValues:
//
//	lazy.SetRegistry(reg)
//	SyncsTotal.Register()
//	for _, result := range []string{"success", "error"} {
//	    SyncResultsTotal.WithLabelValues(result)
//	}
//
// Leave labels with unbounded or externally supplied values to first use.
package lazy
