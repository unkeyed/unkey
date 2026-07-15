-- Per-day rollup of instance_checkpoints_v1 (DASHBOARDS ONLY).
--
-- WARNING: MVs fire on every raw insert, before ReplacingMergeTree dedupe.
-- sample_count double-counts duplicates (rare retries); min/max are
-- idempotent. Billing must hit the raw table with FINAL, never this one.
--
-- Reads raw directly (no chaining) to avoid multi-stage merge latency.
-- Keeps 365 days so month-over-month trend charts work without hitting
-- the minute/hour MVs' shorter TTLs.
CREATE TABLE instance_resources_per_day_v1 (
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
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 365 DAY DELETE;

CREATE MATERIALIZED VIEW instance_resources_per_day_mv_v1
TO instance_resources_per_day_v1 AS
SELECT
  toStartOfDay(fromUnixTimestamp64Milli(ts)) AS time,
  workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id,
  min(cpu_usage_usec) AS cpu_usage_usec_min,
  max(cpu_usage_usec) AS cpu_usage_usec_max,
  max(memory_bytes) AS memory_bytes_max,
  max(cpu_allocated_millicores) AS cpu_allocated_millicores_max,
  max(memory_allocated_bytes) AS memory_allocated_bytes_max,
  max(disk_allocated_bytes) AS disk_allocated_bytes_max,
  max(disk_used_bytes) AS disk_used_bytes_max,
  min(network_egress_public_bytes) AS network_egress_public_bytes_min,
  max(network_egress_public_bytes) AS network_egress_public_bytes_max,
  min(network_egress_private_bytes) AS network_egress_private_bytes_min,
  max(network_egress_private_bytes) AS network_egress_private_bytes_max,
  min(network_ingress_public_bytes) AS network_ingress_public_bytes_min,
  max(network_ingress_public_bytes) AS network_ingress_public_bytes_max,
  min(network_ingress_private_bytes) AS network_ingress_private_bytes_min,
  max(network_ingress_private_bytes) AS network_ingress_private_bytes_max,
  toInt64(count()) AS sample_count
FROM instance_checkpoints_v1
GROUP BY time, workspace_id, project_id, environment_id, resource_type, resource_id, container_uid, instance_id;
