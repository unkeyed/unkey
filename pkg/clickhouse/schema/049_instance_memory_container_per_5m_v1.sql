-- Backfill precedes the MV, so replays skip existing aggregate keys and only the open 5-minute bucket can be missed during handoff.
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

INSERT INTO instance_memory_container_per_5m_v1
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
WHERE instance_resources_per_minute_v1.time >= now() - INTERVAL 7 DAY
  AND instance_resources_per_minute_v1.time < toStartOfInterval(now(), INTERVAL 5 MINUTE)
  AND (workspace_id, project_id, app_id, environment_id, instance_id, container_uid, toStartOfInterval(instance_resources_per_minute_v1.time, INTERVAL 5 MINUTE)) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, instance_id, container_uid, time
    FROM instance_memory_container_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid;

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
