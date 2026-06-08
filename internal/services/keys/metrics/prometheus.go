/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains Key-Verification-related metrics for tracking what keys do.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by type and outcome.
	// The type should be either "root_key" or "key"
	// Use this counter to monitor API traffic patterns.
	//
	// Emission is owned by keys.Get via a deferred increment, so every caller
	// gets the counter for free without remembering to flush. The trade-off is
	// that statuses set later during KeyVerifier.Verify (FORBIDDEN,
	// INSUFFICIENT_PERMISSIONS, RATE_LIMITED, USAGE_EXCEEDED) are recorded here
	// as VALID; use the key_verifications ClickHouse stream for the final outcome.
	//
	// Example usage:
	//   metrics.KeyVerificationsTotal.WithLabelValues("root_key", "VALID").Inc()
	KeyVerificationsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "key",
			Name:      "verifications_total",
			Help:      "Total number of Key verifications processed.",
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
	KeyVerificationErrorsTotal = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "key",
			Name:      "verification_errors_total",
			Help:      "Total number of key verification errors",
		},
		[]string{"type"},
	)
)
