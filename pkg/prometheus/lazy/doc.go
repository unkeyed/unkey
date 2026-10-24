// Package lazy provides lazy-registering Prometheus metric wrappers.
//
// Metrics are declared as package-level globals (like promauto). Each service
// calls [SetRegistry] at startup to control where metrics end up.
//
// # Series exist from startup
//
// A series that does not exist yet is missing, not zero: rate() and
// increase() ignore its first increment, error-ratio alerts evaluate to no
// data, and dashboards show gaps. So [SetRegistry] exports every counter and
// histogram at 0:
//
//   - [Counter] and [Histogram] are exported at 0.
//   - [CounterVec] and [HistogramVec] take their labels as [Values], for a
//     closed set of values, or [Dynamic], for values known only at runtime.
//     When every label has values, every combination is exported at 0.
//   - For a dynamic label, call [CounterVec.Bind] when the owner of the value
//     is constructed, such as a cache with its resource name. That creates
//     the series at 0 before the first event.
//   - Gauges are not exported at 0: their value is state, and 0 can be wrong,
//     for example for a timestamp. Their owner sets them.
//
// Declaring a metric:
//
//	var SyncsTotal = lazy.NewCounterVec(prometheus.CounterOpts{
//	    Namespace: "unkey",
//	    Subsystem: "krane",
//	    Name:      "syncs_total",
//	},
//	    lazy.Dynamic("resource"),
//	    lazy.Values("result", "success", "error"),
//	)
//
// Values only decide which series exist at startup. Other values can still be
// used at runtime.
//
// # Hot paths
//
// Call Bind once when the label values are known, such as in a constructor,
// and update the bound child. WithLabelValues hashes the labels and takes the
// vector's read lock on every call, and that lock contends across cores.
package lazy
