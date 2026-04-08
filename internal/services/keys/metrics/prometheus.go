/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains Key-Verification-related metrics for tracking what keys do.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the key verification system.
type Metrics struct {
	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by type and outcome.
	// The type should be either "root_key" or "key"
	KeyVerificationsTotal *prometheus.CounterVec

	// KeyVerificationErrorsTotal tracks the number of errors in key verifications.
	// These are not errors in the keys themselves like "FORBIDDEN", or "RATE_LIMITED" but errors in
	// program functionality.
	KeyVerificationErrorsTotal *prometheus.CounterVec
}

// NewMetrics creates a new Metrics instance, registering all collectors with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	factory := promauto.With(reg)

	return &Metrics{
		KeyVerificationsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "key",
				Name:      "verifications_total",
				Help:      "Total number of Key verifications processed.",
			},
			[]string{"type", "code"},
		),
		KeyVerificationErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "key",
				Name:      "verification_errors_total",
				Help:      "Total number of key verification errors",
			},
			[]string{"type"},
		),
	}
}
