-- Per-month rollup of instance_checkpoints_v1 (BILLING + DASHBOARDS).
--
-- WARNING: MVs fire on every raw insert, before ReplacingMergeTree dedupe.
-- sample_count double-counts duplicates (rare retries); min/max are
-- idempotent. Billing must hit the raw table with FINAL, never this one.
--
-- Reads raw directly (no chaining) to avoid multi-stage merge latency.
-- Partitions by year (one partition covers 12 rows per container) so
-- long-range queries (year-over-year) hit a small number of parts.
-- Retention kept at 5 years for audit / long-term customer history;
-- bump via ALTER TABLE MODIFY TTL if we need more.
CREATE TABLE instance_resources_per_month_v1 (
  time DateTime,
  workspace_id String,
  project_id LowCardinality(String),
  environment_id LowCardinality(String),
  resource_type LowCardinality(String),
  resource_id LowCardinality(String),
  container_uid String,
  instance_id LowCardinality(String),
  cpu_usage_usec_min SimpleAggregateFunction(min, Int64),
  cpu_usage_usec_max SimpleAggregateFunction(max, Int64),
  memory_bytes_max SimpleAggregateFunction(max, Int64),
  cpu_allocated_millicores_max SimpleAggregateFunction(max, Int32),
  memory_allocated_bytes_max SimpleAggregateFunction(max, Int64),
  disk_allocated_bytes_max SimpleAggregateFunction(max, Int64),
  disk_used_bytes_max SimpleAggregateFunction(max, Int64),
  network_egress_public_bytes_min SimpleAggregateFunction(min, Int64),
  network_egress_public_bytes_max SimpleAggregateFunction(max, Int64),
  network_egress_private_bytes_min SimpleAggregateFunction(min, Int64),
  network_egress_private_bytes_max SimpleAggregateFunction(max, Int64),
  network_ingress_public_bytes_min SimpleAggregateFunction(min, Int64),
  network_ingress_public_bytes_max SimpleAggregateFunction(max, Int64),
  network_ingress_private_bytes_min SimpleAggregateFunction(min, Int64),
  network_ingress_private_bytes_max SimpleAggregateFunction(max, Int64),
  sample_count SimpleAggregateFunction(sum, Int64),
  -- Replica-scoped dashboard filter (web resources.ts). instance_id is not
  -- in the PK; without this, charts filtered to a single replica scan every
  -- container_uid in the (workspace, resource) range.
  INDEX idx_instance_id instance_id TYPE bloom_filter(0.001) GRANULARITY 1
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYY(time)
TTL time + INTERVAL 1825 DAY DELETE;
