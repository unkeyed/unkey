/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains a metric for tracking panics across http handlers.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PanicsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "internal",
		Name:      "panics_total",
		Help:      "Counter to track panics across http handlers",
	}, []string{"caller", "path"})
)
