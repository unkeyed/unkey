// Package metrics declares the Prometheus instruments emitted by
// svc/logdrain. All instruments are lazy — they buffer writes until the
// service registers a registry via lazy.SetRegistry, which keeps the
// package importable from tests and short-lived utilities without
// pulling in a global side-effecting registration.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var (
	// ClickHouseQueryDuration is the wall-clock latency of one CH SELECT
	// per (source, shard) — i.e., one cursor query for one group on this
	// pod. Used to spot CH-side regressions before they show up as lag.
	ClickHouseQueryDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "logdrain",
			Subsystem: "clickhouse",
			Name:      "query_duration_seconds",
			Help:      "Histogram of ClickHouse query latencies by group source and shard.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"source", "shard"},
	)

	// ClickHouseQueryErrors counts CH SELECT failures by classifier.
	ClickHouseQueryErrors = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "clickhouse",
			Name:      "query_errors_total",
			Help:      "Total ClickHouse query errors by error type.",
		},
		[]string{"error_type"},
	)

	// ClickHouseRecordsFetched counts rows read from CH per source.
	// workspace_id was intentionally not added because it grows
	// cumulatively past Prometheus's comfortable cardinality envelope;
	// per-tenant numbers go through structured logs instead.
	ClickHouseRecordsFetched = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "clickhouse",
			Name:      "records_fetched_total",
			Help:      "Total records fetched from ClickHouse, by source.",
		},
		[]string{"source"},
	)

	// ActiveGroups is the count of (workspace, project, env, source)
	// groups currently being processed by this pod.
	ActiveGroups = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "logdrain",
			Subsystem: "coordinator",
			Name:      "active_groups",
			Help:      "Number of active drain groups being processed by this pod.",
		},
		[]string{"shard"},
	)

	// GroupsSkippedLimit counts groups dropped because the shard exceeds
	// MaxGroupsPerShard. Non-zero means the shard is over capacity and
	// the rotating window is in effect.
	GroupsSkippedLimit = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "coordinator",
			Name:      "groups_skipped_limit_total",
			Help:      "Groups skipped due to MaxGroupsPerShard.",
		},
		[]string{"shard"},
	)

	// TickDuration is end-to-end coordinator tick time. p99 climbing
	// toward poll_interval is the early signal that the pod is at
	// capacity.
	TickDuration = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "logdrain",
			Subsystem: "coordinator",
			Name:      "tick_duration_seconds",
			Help:      "Histogram of coordinator tick processing times.",
			Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30},
		},
	)

	// CursorUpdateDeadlocks counts optimistic-lock losses on
	// AdvanceLogDrainCursor. group_key is intentionally not a label
	// because the counter is cumulative and one bursty group can
	// generate thousands of permanent label values.
	CursorUpdateDeadlocks = lazy.NewCounter(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "cursor",
			Name:      "update_deadlocks_total",
			Help:      "MySQL cursor update deadlocks (optimistic locking failures).",
		},
	)

	// CursorAdvanceLatency is the wall-clock latency of the per-drain
	// cursor UPDATE. Tail latencies climbing here usually mean MySQL is
	// the bottleneck, not CH or the provider.
	CursorAdvanceLatency = lazy.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "logdrain",
			Subsystem: "cursor",
			Name:      "advance_latency_seconds",
			Help:      "Latency of cursor advance operations.",
			Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5},
		},
	)

	// GroupLagSeconds is the per-group lag between CH watermark and the
	// group's MIN drain cursor. Cardinality is the active-group count
	// (not cumulative), so a labelled gauge is safe here.
	GroupLagSeconds = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "logdrain",
			Subsystem: "cursor",
			Name:      "group_lag_seconds",
			Help:      "Lag between ClickHouse watermark and cursor position, per group.",
		},
		[]string{"group_key"},
	)

	// ProviderErrors counts failed sink.Send calls. Aggregated across
	// workspaces; per-tenant breakdowns live in logs.
	ProviderErrors = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "provider",
			Name:      "errors_total",
			Help:      "Provider delivery errors by error type, aggregated across workspaces.",
		},
		[]string{"provider", "error_type"},
	)

	// ProviderRequestDuration is the wall-clock latency of one HTTP
	// round-trip to a provider, including retries.
	ProviderRequestDuration = lazy.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "logdrain",
			Subsystem: "provider",
			Name:      "request_duration_seconds",
			Help:      "Provider HTTP request latency.",
			Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30},
		},
		[]string{"provider"},
	)

	// RecordsDelivered counts successful per-record deliveries.
	RecordsDelivered = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "provider",
			Name:      "records_delivered_total",
			Help:      "Records successfully delivered to providers.",
		},
		[]string{"provider"},
	)

	// RecordsBytesDelivered counts gzip-compressed bytes shipped over
	// the wire — useful for sizing egress and provider plan tier.
	RecordsBytesDelivered = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "provider",
			Name:      "bytes_delivered_total",
			Help:      "Compressed bytes delivered to providers.",
		},
		[]string{"provider"},
	)

	// CredentialDecrypts counts vault calls behind the credential cache.
	CredentialDecrypts = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "credentials",
			Name:      "decrypts_total",
			Help:      "Credential decryption attempts by result.",
		},
		[]string{"result"}, // success, vault_error, not_found
	)

	// CredentialCacheHits tracks cache effectiveness; "miss" should be
	// rare in steady state because a drain ticks 6× per minute.
	CredentialCacheHits = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "credentials",
			Name:      "cache_hits_total",
			Help:      "Credential cache hit/miss stats.",
		},
		[]string{"result"}, // hit, miss
	)

	// OAuthGrantRevoked increments on the first 401 from a provider
	// for a grant — every drain referencing that grant flips paused on
	// the same tick.
	OAuthGrantRevoked = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "oauth",
			Name:      "grant_revoked_total",
			Help:      "OAuth grants revoked by provider.",
		},
		[]string{"provider"},
	)

	// OAuthTokenRefresh counts token refresh outcomes.
	OAuthTokenRefresh = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "oauth",
			Name:      "token_refresh_total",
			Help:      "OAuth token refresh attempts by result.",
		},
		[]string{"provider", "result"}, // success, failure
	)

	// DrainsAutopaused increments when consecutive_failures crosses the
	// auto-pause threshold.
	DrainsAutopaused = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "drain",
			Name:      "autopaused_total",
			Help:      "Drains auto-paused due to repeated failures.",
		},
		[]string{"provider", "reason"},
	)

	// DrainsResumed counts manual resumes from the dashboard.
	DrainsResumed = lazy.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logdrain",
			Subsystem: "drain",
			Name:      "resumed_total",
			Help:      "Drains manually resumed after auto-pause.",
		},
		[]string{"provider"},
	)

	// GroupsBlockedByDrain is the number of groups in this shard that
	// have at least one drain in `blocked` state. Per-shard rather than
	// per-(drain, group) because the latter pair is cumulative
	// cardinality on a gauge.
	GroupsBlockedByDrain = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "logdrain",
			Subsystem: "drain",
			Name:      "groups_blocked",
			Help:      "Groups currently blocked by a paused drain, per shard.",
		},
		[]string{"shard"},
	)

	// MemoryUsageBytes reports component-level RSS so we can size
	// limits without guessing.
	MemoryUsageBytes = lazy.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "logdrain",
			Subsystem: "memory",
			Name:      "usage_bytes",
			Help:      "Memory usage by component (credential_cache, batch_buffers, ...).",
		},
		[]string{"component"},
	)
)
