CREATE TABLE instance_resources_container_per_5m_v1 (
  time DateTime,
  workspace_id String,
  anomaly_shard UInt8 MATERIALIZED cityHash64(workspace_id) % 256,
  project_id LowCardinality(String),
  app_id LowCardinality(String),
  environment_id LowCardinality(String),
  instance_id LowCardinality(String),
  container_uid String,
  cpu_usage_usec_min SimpleAggregateFunction(min, Int64),
  cpu_usage_usec_max SimpleAggregateFunction(max, Int64),
  network_egress_public_bytes_min SimpleAggregateFunction(min, Int64),
  network_egress_public_bytes_max SimpleAggregateFunction(max, Int64),
  utilization_sum SimpleAggregateFunction(sum, Float64),
  utilization_samples SimpleAggregateFunction(sum, UInt64),
  utilization_max SimpleAggregateFunction(max, Float64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (anomaly_shard, workspace_id, project_id, app_id, environment_id, instance_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW instance_resources_container_per_5m_mv_v1
TO instance_resources_container_per_5m_v1 AS
SELECT
  toStartOfInterval(fromUnixTimestamp64Milli(ts), INTERVAL 5 MINUTE) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  instance_id,
  container_uid,
  min(cpu_usage_usec) AS cpu_usage_usec_min,
  max(cpu_usage_usec) AS cpu_usage_usec_max,
  min(network_egress_public_bytes) AS network_egress_public_bytes_min,
  max(network_egress_public_bytes) AS network_egress_public_bytes_max,
  sumIf(toFloat64(memory_bytes) / toFloat64(memory_allocated_bytes), memory_allocated_bytes > 0) AS utilization_sum,
  toUInt64(countIf(memory_allocated_bytes > 0)) AS utilization_samples,
  maxIf(toFloat64(memory_bytes) / toFloat64(memory_allocated_bytes), memory_allocated_bytes > 0) AS utilization_max
FROM instance_checkpoints_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid;
