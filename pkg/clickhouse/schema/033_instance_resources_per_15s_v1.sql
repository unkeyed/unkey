-- Per-15-second rollup of instance_checkpoints_v1 (DASHBOARDS ONLY).
-- Granular history for short chart windows (≤ 1 hour). See 027 for the
-- dedupe caveat — NEVER use this table for billing.
--
-- Reads raw directly (flat cascade). ~4× the row count of per-minute per
-- container, so TTL is tighter at 7 days. Charts requesting windows
-- longer than 7 days fall back to per_minute or per_hour.
CREATE TABLE instance_resources_per_15s_v1 (
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
  memory_bytes_sum SimpleAggregateFunction(sum, Int64),
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
TTL time + INTERVAL 7 DAY DELETE;
