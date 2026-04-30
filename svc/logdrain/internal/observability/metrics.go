// Package observability provides comprehensive metrics and monitoring for
// logdrain service components including query latency, cursor progression,
// provider error rates, and resource usage tracking.
package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ClickHouse query metrics
	ClickHouseQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "logdrain",
		Subsystem: "clickhouse",
		Name:      "query_duration_seconds",
		Help:      "Histogram of ClickHouse query latencies by group",
		Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"source", "shard"})

	ClickHouseQueryErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "clickhouse",
		Name:      "query_errors_total",
		Help:      "Total ClickHouse query errors by error type",
	}, []string{"error_type"})

	ClickHouseRecordsFetched = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "clickhouse",
		Name:      "records_fetched_total",
		Help:      "Total records fetched from ClickHouse",
	}, []string{"source", "workspace_id"})

	// Coordinator metrics
	ActiveGroups = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "logdrain",
		Subsystem: "coordinator",
		Name:      "active_groups",
		Help:      "Number of active drain groups being processed",
	}, []string{"shard"})

	GroupsSkippedLimit = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "coordinator",
		Name:      "groups_skipped_limit_total",
		Help:      "Groups skipped due to shard limits",
	}, []string{"shard"})

	TickDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "logdrain",
		Subsystem: "coordinator",
		Name:      "tick_duration_seconds",
		Help:      "Histogram of coordinator tick processing times",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30},
	})

	// Cursor management metrics  
	CursorUpdateDeadlocks = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "cursor",
		Name:      "update_deadlocks_total",
		Help:      "MySQL cursor update deadlocks (optimistic locking failures)",
	}, []string{"group_key"})

	CursorAdvanceLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "logdrain",
		Subsystem: "cursor",
		Name:      "advance_latency_seconds",
		Help:      "Latency of cursor advance operations",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
	})

	GroupLagSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "logdrain",
		Subsystem: "cursor",
		Name:      "group_lag_seconds",
		Help:      "Lag between ClickHouse inserted_at and cursor position",
	}, []string{"group_key"})

	// Provider error tracking - separated for cardinality management
	ProviderErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "provider",
		Name:      "errors_total",
		Help:      "Provider delivery errors by type (aggregated across workspaces)",
	}, []string{"provider", "error_type"})

	// High-cardinality customer-specific metrics (use sparingly for debugging)
	ProviderErrorsByWorkspace = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "provider",
		Name:      "errors_by_workspace_total",
		Help:      "Provider errors by workspace (high cardinality - for customer debugging)",
	}, []string{"provider", "workspace_id"})

	ProviderRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "logdrain",
		Subsystem: "provider",
		Name:      "request_duration_seconds",
		Help:      "Provider HTTP request latency (aggregated)",
		Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
	}, []string{"provider"})

	// Aggregated delivery metrics (low cardinality)
	RecordsDelivered = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "provider",
		Name:      "records_delivered_total",
		Help:      "Total records successfully delivered to providers (aggregated)",
	}, []string{"provider"})

	RecordsBytesDelivered = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "provider",
		Name:      "bytes_delivered_total",
		Help:      "Total bytes delivered to providers (aggregated)",
	}, []string{"provider"})

	// Customer-specific delivery metrics (for support/debugging) 
	WorkspaceDeliverySuccess = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "workspace",
		Name:      "delivery_success_total", 
		Help:      "Successful deliveries by workspace (high cardinality)",
	}, []string{"workspace_id"})

	// Credential and authentication metrics
	CredentialDecrypts = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "credentials",
		Name:      "decrypts_total",
		Help:      "Credential decryption attempts by result",
	}, []string{"result"}) // success, vault_error, not_found

	CredentialCacheHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "credentials",
		Name:      "cache_hits_total",
		Help:      "Credential cache hit/miss stats",
	}, []string{"result"}) // hit, miss

	OAuthGrantRevoked = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "oauth",
		Name:      "grant_revoked_total",
		Help:      "OAuth grants revoked by provider",
	}, []string{"provider"})

	OAuthTokenRefresh = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "oauth",
		Name:      "token_refresh_total",
		Help:      "OAuth token refresh attempts by result",
	}, []string{"provider", "result"}) // success, failure

	// Drain lifecycle metrics
	DrainsAutopaused = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "drain",
		Name:      "autopaused_total",
		Help:      "Drains auto-paused due to repeated failures",
	}, []string{"provider", "reason"})

	DrainsResumed = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "logdrain",
		Subsystem: "drain",
		Name:      "resumed_total",
		Help:      "Drains manually resumed after auto-pause",
	}, []string{"provider"})

	GroupsBlockedByDrain = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "logdrain",
		Subsystem: "drain",
		Name:      "groups_blocked_by_drain",
		Help:      "Number of groups blocked by a single paused drain",
	}, []string{"drain_id", "group_key"})

	// Memory and resource metrics
	MemoryUsageBytes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "logdrain",
		Subsystem: "memory",
		Name:      "usage_bytes",
		Help:      "Memory usage by component",
	}, []string{"component"}) // credential_cache, batch_buffers, etc

	BatchBufferSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "logdrain",
		Subsystem: "memory",
		Name:      "batch_buffer_records",
		Help:      "Number of records held in memory for processing",
	}, []string{"group_key"})
)

// ErrorType categorizes provider errors for metrics
func ErrorType(err error) string {
	// TODO: Implement error classification logic
	// auth, rate_limit, timeout, 5xx, 4xx, network
	return "unknown"
}
