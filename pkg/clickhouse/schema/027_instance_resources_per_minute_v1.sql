-- Per-minute rollup of instance_checkpoints_v1 (DASHBOARDS ONLY).
--
-- WARNING: MVs fire on every raw insert, before ReplacingMergeTree dedupe.
-- *_sum and sample_count double-count duplicates (rare retries). min/max are
-- idempotent. Billing must hit the raw table with FINAL, never this one.
--
-- Reads raw directly (no chaining) to avoid multi-stage merge latency.
CREATE TABLE instance_resources_per_minute_v1 (
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
  sample_count SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (workspace_id, resource_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;
