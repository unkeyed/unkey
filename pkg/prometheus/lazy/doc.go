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
package lazy
