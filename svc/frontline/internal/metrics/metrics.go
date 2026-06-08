package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// RequestsTotal is the unified per-request outcome counter. One row per
// completed request, labelled by:
//
//	status_class 2xx | 3xx | 4xx | 5xx
//	fault_domain "" on success, else the attribution domain — platform |
//	             routing | capacity | upstream | client. This is the segment
//	             that drives alert routing: alerts select fault_domain="..."
//	             instead of regex-matching the URN. It is the category segment
//	             of the fault URN (see fault.GetCategory).
//	code         "" on success, the fault URN string on error
//	             (e.g. err:frontline:routing:config_not_found). High-fidelity
//	             identifier for dashboards / triage breakdowns.
//
// `code != ""` is the success/error split — there is no separate outcome
// label, the URN carries that bit. region and environment are external labels
// on the per-region Prometheus and are NOT carried as metric labels.
var RequestsTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "requests_total",
		Help:      "Total requests by status class, attribution domain, and fault code (URN).",
	},
	[]string{"status_class", "fault_domain", "code"},
)

// InflightRequests is the per-pod count of in-flight requests. Incremented
// on entry to the observability middleware and decremented on exit, so it
// covers every routed path.
//
// Saturation signal: if the gauge climbs while traffic is steady, an
// upstream got slow or frontline stalled. Pair with goroutine count for
// triage — a leak shows up in both.
var InflightRequests = lazy.NewGauge(
	prometheus.GaugeOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "inflight_requests",
		Help:      "Requests currently being processed.",
	},
)

// StatusClass returns the HTTP status class label for a status code:
// "2xx", "3xx", "4xx", or "5xx".
func StatusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	default:
		return "2xx"
	}
}
