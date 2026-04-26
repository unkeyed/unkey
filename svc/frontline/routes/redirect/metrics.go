package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

// redirectsTotal counts plain-HTTP requests upgraded to HTTPS via the
// HTTP-listener catchall. Labels are intentionally omitted to keep the
// hot path allocation-free; per-rule-kind metrics live on the HTTPS
// listener via edgeredirect.EdgeRedirectsTotal.
var redirectsTotal = lazy.NewCounter(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "https_redirects_total",
		Help:      "Total number of plain-HTTP requests upgraded to HTTPS via the HTTP listener.",
	},
)
