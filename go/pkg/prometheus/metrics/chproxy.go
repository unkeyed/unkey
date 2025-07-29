/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains ClickHouse proxy-related metrics for tracking event ingestion.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ChproxyRequestsTotal tracks the total number of chproxy requests received, labeled by endpoint.
	// Use this counter to monitor ingestion traffic patterns.
	//
	// Example usage:
	//   metrics.ChproxyRequestsTotal.WithLabelValues("verifications").Inc()
	ChproxyRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "chproxy",
			Name:        "requests_total",
			Help:        "Total number of ClickHouse proxy requests processed.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)

	// ChproxyRowsTotal tracks the total number of rows/events received in chproxy requests.
	// Use this counter to monitor data volume and ingestion patterns.
	//
	// Example usage:
	//   metrics.ChproxyRowsTotal.WithLabelValues("verifications").Add(float64(len(events)))
	ChproxyRowsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "chproxy",
			Name:        "rows_total",
			Help:        "Total number of rows/events processed by ClickHouse proxy.",
			ConstLabels: constLabels,
		},
		[]string{"endpoint"},
	)
)
