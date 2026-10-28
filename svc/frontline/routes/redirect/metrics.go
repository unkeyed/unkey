package handler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// redirectsTotal counts plain-HTTP requests upgraded to HTTPS via 308.
// Labels are intentionally omitted to keep the hot path allocation-free.
var redirectsTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "frontline",
		Name:      "https_redirects_total",
		Help:      "Total number of plain-HTTP requests upgraded to HTTPS via 308.",
	},
)
