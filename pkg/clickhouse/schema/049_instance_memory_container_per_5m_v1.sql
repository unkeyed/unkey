-- Memory keeps container and instance grain so the detector can average time
-- within each container, then containers within each instance, then instances
-- within the app. A direct app average would over-weight instances with more
-- containers or samples.
CREATE TABLE instance_memory_container_per_5m_v1 (
  time DateTime,
  workspace_id String,
  anomaly_shard UInt8 MATERIALIZED cityHash64(workspace_id) % 256,
  project_id LowCardinality(String),
  app_id LowCardinality(String),
  environment_id LowCardinality(String),
  instance_id LowCardinality(String),
  container_uid String,
  utilization_sum SimpleAggregateFunction(sum, Float64),
  utilization_samples SimpleAggregateFunction(sum, UInt64),
  utilization_max SimpleAggregateFunction(max, Float64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (anomaly_shard, workspace_id, project_id, app_id, environment_id, instance_id, container_uid, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW instance_memory_container_per_5m_mv_v1
TO instance_memory_container_per_5m_v1 AS
SELECT
  toStartOfInterval(time, INTERVAL 5 MINUTE) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  instance_id,
  container_uid,
  sumIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS utilization_sum,
  toUInt64(countIf(memory_allocated_bytes_max > 0)) AS utilization_samples,
  maxIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS utilization_max
FROM instance_resources_per_minute_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid;
