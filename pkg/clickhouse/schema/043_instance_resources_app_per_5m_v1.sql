CREATE TABLE instance_resources_app_per_5m_v1 (
  time DateTime,
  workspace_id String,
  anomaly_shard UInt8 MATERIALIZED cityHash64(workspace_id) % 256,
  project_id LowCardinality(String),
  app_id LowCardinality(String),
  environment_id LowCardinality(String),
  egress_bytes SimpleAggregateFunction(sum, Int64),
  cpu_seconds SimpleAggregateFunction(sum, Float64),
  memory_utilization_sum SimpleAggregateFunction(sum, Float64),
  memory_utilization_samples SimpleAggregateFunction(sum, UInt64),
  memory_utilization_max SimpleAggregateFunction(max, Float64),
  instance_ids AggregateFunction(uniqCombined64, String)
)
ENGINE = AggregatingMergeTree()
ORDER BY (anomaly_shard, workspace_id, project_id, app_id, environment_id, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

CREATE MATERIALIZED VIEW instance_resources_app_per_5m_mv_v1
TO instance_resources_app_per_5m_v1 AS
SELECT
  toStartOfInterval(time, INTERVAL 5 MINUTE) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sum(greatest(toInt64(0), network_egress_public_bytes_max - network_egress_public_bytes_min)) AS egress_bytes,
  sum(toFloat64(greatest(toInt64(0), cpu_usage_usec_max - cpu_usage_usec_min))) / 1e6 AS cpu_seconds,
  sumIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization_sum,
  toUInt64(countIf(memory_allocated_bytes_max > 0)) AS memory_utilization_samples,
  maxIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization_max,
  uniqCombined64State(instance_id) AS instance_ids
FROM instance_resources_per_minute_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id;

INSERT INTO instance_resources_app_per_5m_v1
SELECT
  toStartOfInterval(time, INTERVAL 5 MINUTE) AS time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sum(greatest(toInt64(0), network_egress_public_bytes_max - network_egress_public_bytes_min)) AS egress_bytes,
  sum(toFloat64(greatest(toInt64(0), cpu_usage_usec_max - cpu_usage_usec_min))) / 1e6 AS cpu_seconds,
  sumIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization_sum,
  toUInt64(countIf(memory_allocated_bytes_max > 0)) AS memory_utilization_samples,
  maxIf(toFloat64(memory_bytes_max) / toFloat64(memory_allocated_bytes_max), memory_allocated_bytes_max > 0) AS memory_utilization_max,
  uniqCombined64State(instance_id) AS instance_ids
FROM instance_resources_per_minute_v1
WHERE instance_resources_per_minute_v1.time >= now() - INTERVAL 7 DAY
  AND (workspace_id, project_id, app_id, environment_id, toStartOfInterval(instance_resources_per_minute_v1.time, INTERVAL 5 MINUTE)) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, time
    FROM instance_resources_app_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id;
