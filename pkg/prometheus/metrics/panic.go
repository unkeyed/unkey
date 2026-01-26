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
	// PanicsTotal tracks panics recovered by HTTP handler middleware.
	// Use this counter to monitor application stability and identify handlers
	// that are prone to panicking.
	//
	// Labels:
	//   - "caller": The function or handler that panicked
	//   - "path": The HTTP request path that triggered the panic
	//
	// Example usage:
	//   metrics.PanicsTotal.WithLabelValues("handleVerifyKey", "/v1/keys.verifyKey").Inc()
	PanicsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace:   "unkey",
		Subsystem:   "internal",
		Name:        "panics_total",
		Help:        "Total number of panics recovered in HTTP handlers.",
		ConstLabels: constLabels,
	}, []string{"caller", "path"})
)
