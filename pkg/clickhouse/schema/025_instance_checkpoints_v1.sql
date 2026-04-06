-- Billing query rules:
--   1. Use FINAL. ReplacingMergeTree dedupe runs on merges, not inserts, so
--      un-merged duplicate rows are visible to plain queries. Pair-integration
--      over duplicates is NOT idempotent.
--      At scale, also set:
--        SETTINGS do_not_merge_across_partitions_select_final = 1
--      or query performance collapses as partition count grows.
--   2. Group by container_uid. Never mix counter values across a container
--      restart boundary. container_uid = pod_uid + restart_count.
--
-- Prefer the `instance_checkpoints` VIEW over this table directly — the view
-- already applies FINAL.
--
-- CPU usage (seconds):
--   (max(cpu_usage_usec) - min(cpu_usage_usec)) / 1e6
--
-- Memory byte-seconds:
--   sum(least(memory_bytes, leadInFrame(memory_bytes)) *
--       (leadInFrame(ts) - ts)) / 1000
--
-- Disk byte-seconds:
--   max(disk_allocated_bytes) * (max(ts) - min(ts)) / 1000
--
-- Network byte totals (same shape as CPU):
--   max(network_*_bytes) - min(network_*_bytes)
--
-- Utilization % (observability, not billing):
--   used_cpu_seconds / (cpu_allocated_millicores / 1000 * window_seconds)
--   max(memory_bytes) / max(memory_allocated_bytes)

CREATE TABLE instance_checkpoints_v1 (
  node_id LowCardinality(String),
  workspace_id String,
  project_id LowCardinality(String),
  environment_id LowCardinality(String),
  resource_type LowCardinality(String),
  resource_id LowCardinality(String),
  pod_uid String,
  instance_id String,
  container_uid String,
  restart_count UInt32 CODEC(T64, ZSTD(1)),
  -- Unix milliseconds from a monotonic-anchored clock. Intervals never
  -- regress, so memory pair-integration dt is always non-negative.
  ts Int64 CODEC(Delta, ZSTD(1)),
  event_kind LowCardinality(String),

  -- Monotonic CPU usage counter (microseconds) from cgroup v2
  -- cpu.stat:usage_usec. Does NOT include throttled time.
  -- Int64 (not UInt64): if a billing query ever computes max-min without
  -- clamping, UInt64 would underflow to ~1.8e19 (584k CPU-years). Int64
  -- produces a visible negative instead.
  -- Monotonic per container_uid: Delta+ZSTD compresses ~50x.
  cpu_usage_usec Int64 CODEC(Delta, ZSTD(1)),

  -- Working-set memory (bytes): memory.current - memory.stat:inactive_file.
  -- Matches kubelet's container_memory_working_set_bytes; excludes
  -- reclaimable page cache so we don't bill customers for kernel cache.
  -- Slow-moving gauge: DoubleDelta+ZSTD.
  memory_bytes Int64 CODEC(DoubleDelta, ZSTD(1)),

  -- Pod's declared CPU/memory (millicores, bytes). Comes from pod.Spec on
  -- every checkpoint so resize events are naturally observable as value
  -- changes over time. Used for utilization% dashboards. Not billed.
  -- Nearly-constant per container_uid: DoubleDelta collapses to almost nothing.
  cpu_allocated_millicores Int32 CODEC(DoubleDelta, ZSTD(1)),
  memory_allocated_bytes Int64 CODEC(DoubleDelta, ZSTD(1)),

  -- Allocated PVC size (bytes). Assumed constant per container_uid.
  disk_allocated_bytes Int64 CODEC(DoubleDelta, ZSTD(3)),
  -- Actual used bytes on CSI-backed ephemeral volumes (statfs). Observational.
  disk_used_bytes Int64 CODEC(DoubleDelta, ZSTD(1)),

  -- Network byte counters, monotonic per container_uid. Reserved for a
  -- future eBPF/Hubble collector — currently always 0.
  -- Billing: max(counter) - min(counter). Delta+ZSTD like CPU.
  network_egress_public_bytes Int64 CODEC(Delta, ZSTD(1)),
  network_egress_private_bytes Int64 CODEC(Delta, ZSTD(1)),
  network_ingress_public_bytes Int64 CODEC(Delta, ZSTD(1)),
  network_ingress_private_bytes Int64 CODEC(Delta, ZSTD(1)),

  region LowCardinality(String),
  platform LowCardinality(String),

  -- Open-schema diagnostic metadata that doesn't earn a typed column:
  -- image_id, kernel_version, eBPF program version, node_pool, pod label
  -- snapshot, etc. Not propagated to rollups (debug-only); does not bill.
  -- ZSTD on empty `{}` is a few bytes/row.
  attributes JSON CODEC(ZSTD(1)),

  -- Skip indexes for cross-workspace admin and project-scoped queries. The
  -- ORDER BY already serves workspace-scoped billing.
  INDEX idx_project project_id TYPE bloom_filter(0.01) GRANULARITY 4,
  INDEX idx_resource resource_id TYPE bloom_filter(0.01) GRANULARITY 4,
  INDEX idx_ts ts TYPE minmax GRANULARITY 1,
  -- Replica-scoped dashboard filter (web resources.ts). instance_id is the
  -- k8s pod name; not in PK, so without this index a per-replica chart
  -- scans every container_uid in the deployment. 0.001 FP rate because
  -- this filter is the selective one when set.
  INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = ReplacingMergeTree
ORDER BY (workspace_id, container_uid, ts)
-- Daily partitions: monthly would hit ~260B rows/partition at 100k/s tail
-- (unmergeable). Daily caps at ~8.6B rows/partition; ~90 active partitions
-- at 90-day retention is well within ClickHouse's healthy range.
PARTITION BY toYYYYMMDD(fromUnixTimestamp64Milli(ts))
-- 5-day TTL grace: accommodates agent outages up to 5 days where
-- retroactive checkpoints would otherwise be immediately dropped.
TTL toDateTime(fromUnixTimestamp64Milli(ts)) + INTERVAL 95 DAY DELETE
SETTINGS ttl_only_drop_parts = 1;
