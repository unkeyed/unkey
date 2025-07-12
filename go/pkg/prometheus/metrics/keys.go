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
	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by some data.
	// Use this counter to monitor API traffic patterns and error rates.
	//
	// Example usage:
	//   metrics.KeyVerificationsTotal.WithLabelValues("ws_1234", "api_5678", "key_abcd", "true", "VALID").Inc()
	KeyVerificationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem:   "key",
			Name:        "verifications_total",
			Help:        "Total number of Key verifications processed.",
			ConstLabels: constLabels,
		},
		[]string{"workspaceId", "apiId", "keyId", "valid", "code"},
	)

	// KeyCreditsSpentTotal tracks the total credits spent by keys, labeled by workspace ID, key ID, and identity ID.
	// Use this counter to monitor credit usage patterns and error rates.
	//
	// Example usage:
	//   metrics.KeyCreditsSpentTotal.WithLabelValues("ws_1234", "key_abcd", "identity_xyz", "5").Add(5)
	KeyCreditsSpentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "key",
			Name:      "credits_spent_total",
			Help:      "Total credits spent by keys",
		},
		[]string{"workspace_id", "key_id", "identity_id", "deducted"},
	)
)
