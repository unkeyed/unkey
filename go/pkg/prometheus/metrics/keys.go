/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains Key-Verification-related metrics for tracking what keys do.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by type and outcome.
	// The type should be either "root_key" or "key"
	// Use this counter to monitor API traffic patterns.
	//
	// Example usage:
	//   metrics.KeyVerificationsTotal.WithLabelValues("root_key", "VALID").Inc()
	KeyVerificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "key",
			Name:        "verifications_total",
			Help:        "Total number of Key verifications processed.",
			ConstLabels: constLabels,
		},
		[]string{"type", "code"},
	)

	// KeyVerificationErrorsTotal tracks the number of errors in key verifications.
	// These are not errors in the keys themselves like "FORBIDDEN", or "RATE_LIMITED" but errors in
	// program functionality. Use this with the unkey_key_verifications_total metric to calculate
	// the error rate.
	//
	// Example usage:
	//   metrics.KeyVerificationErrorsTotal.WithLabelValues("root_key").Inc()
	KeyVerificationErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace:   "unkey",
			Subsystem:   "key",
			Name:        "verification_errors_total",
			Help:        "Total number of key verification errors",
			ConstLabels: constLabels,
		},
		[]string{"type"},
	)
)
