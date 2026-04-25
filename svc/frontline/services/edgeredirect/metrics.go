package edgeredirect

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// EdgeRedirectsTotal counts redirects served from per-FQDN rules on the
// HTTPS listener, labeled by rule kind. Cardinality is bounded by the
// proto oneof (currently four kinds), safe to label.
//
// The HTTP listener bumps a separate unlabeled
// `unkey_frontline_https_redirects_total` counter so its dashboards stay
// stable across this refactor and so the two listeners do not double-count
// each other on shared tooling.
var EdgeRedirectsTotal = lazy.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "edge_redirects_total",
		Help:      "Total number of edge-redirect rules that fired on the HTTPS listener, labeled by rule kind.",
	},
	[]string{"rule_kind"},
)
