/*
Package metrics provides Prometheus metric collectors for monitoring application performance.

This file contains Key-Verification-related metrics for tracking what keys do.
*/
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/vec"
)

var (
	// AuthIPAllowlistDenialsTotal counts authentication attempts rejected by an
	// IP allow list.
	// Investigate root-key denials in Grafana: search API logs for "This IP address
	// is not allowed to use the root key." and inspect auth.root_key_id, auth.workspace_id,
	// and the request ID. IDs stay in logs to avoid unbounded metric cardinality.
	// The AuthIPAllowlistDenied alert includes a filtered Grafana Logs link.
	AuthIPAllowlistDenialsTotal = vec.NewCounter(prometheus.CounterOpts{
		Namespace: "unkey",
		Subsystem: "auth",
		Name:      "ip_allowlist_denials_total",
		Help:      `Authentication attempts rejected by an IP allowlist. For root-key denials, search API logs in Grafana for "This IP address is not allowed to use the root key." The AuthIPAllowlistDenied alert includes a filtered Logs link.`,
	})

	// KeyVerificationsTotal tracks the number of key verifications handled, labeled by type, outcome,
	// and internal reason.
	// The type should be either "root_key" or "key"
	// Use this counter to monitor API traffic patterns.
	//
	// Every verification is recorded exactly once with its terminal status:
	// keys.Get records the statuses it decides (NOT_FOUND, DISABLED, EXPIRED,
	// WORKSPACE_DISABLED, WORKSPACE_NOT_FOUND), KeyVerifier.Hide records keys
	// that passed Get but are hidden from the caller, and KeyVerifier.Verify
	// records the outcome of every other key that passed Get, including the
	// Verify-stage rejections (FORBIDDEN, INSUFFICIENT_PERMISSIONS,
	// RATE_LIMITED, USAGE_EXCEEDED).
	//
	// The reason label holds the values of keys.Reason. It tells apart the
	// causes that share a code, such as unknown_hash and workspace_mismatch
	// for NOT_FOUND.
	//
	// Example usage:
	//   metrics.KeyVerificationsTotal.WithLabelValues("root_key", "VALID", "none").Inc()
	KeyVerificationsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "key",
			Name:      "verifications_total",
			Help:      "Total number of Key verifications processed.",
		},
		vec.Values("type", "key", "root_key"),
		vec.Values("code",
			"VALID",
			"NOT_FOUND",
			"DISABLED",
			"EXPIRED",
			"FORBIDDEN",
			"INSUFFICIENT_PERMISSIONS",
			"RATE_LIMITED",
			"USAGE_EXCEEDED",
			"WORKSPACE_DISABLED",
			"WORKSPACE_NOT_FOUND",
		),
		vec.Values("reason",
			"none",
			"unknown_hash",
			"migration_not_found",
			"key_deleted",
			"api_deleted",
			"workspace_mismatch",
			"keyspace_mismatch",
			"permission_denied",
			"disabled",
			"expired",
			"ip_denied",
			"workspace_disabled",
			"workspace_not_found",
			"insufficient_permissions",
			"rate_limited",
			"usage_exceeded",
		),
	)

	// KeyVerificationFailureRowsTotal counts the rows offered to the internal
	// key_verification_failures ClickHouse stream, by what the per-pod cap did
	// with them. "recorded" rows were below the cap, "sampled" rows were kept
	// above the cap with a sample weight, and "sampled_out" rows were not
	// written. Rows lost to a full buffer are counted by
	// unkey_buffer_state_total{name="key_verification_failures",state="dropped"}.
	KeyVerificationFailureRowsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "key",
			Name:      "verification_failure_rows_total",
			Help:      "Key verification failure rows offered to ClickHouse, by cap decision.",
		},
		vec.Values("result", "recorded", "sampled", "sampled_out"),
	)

	// KeyVerificationErrorsTotal tracks the number of errors in key verifications.
	// These are not errors in the keys themselves like "FORBIDDEN", or "RATE_LIMITED" but errors in
	// program functionality. Use this with the unkey_key_verifications_total metric to calculate
	// the error rate.
	//
	// Example usage:
	//   metrics.KeyVerificationErrorsTotal.WithLabelValues("root_key").Inc()
	KeyVerificationErrorsTotal = vec.NewCounter(
		prometheus.CounterOpts{
			Namespace: "unkey",
			Subsystem: "key",
			Name:      "verification_errors_total",
			Help:      "Total number of key verification errors",
		},
		vec.Values("type", "key", "root_key"),
	)
)
