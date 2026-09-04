-- Backfills precede their MVs, so replays skip or safely merge existing aggregate keys and only open 5-minute buckets can be missed during handoff.
CREATE TABLE anomaly_source_watermarks_v1 (
  source LowCardinality(String),
  region LowCardinality(String),
  time SimpleAggregateFunction(max, DateTime)
)
ENGINE = AggregatingMergeTree()
ORDER BY (source, region);

INSERT INTO anomaly_source_watermarks_v1
SELECT
  'requests' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM frontline_requests_raw_v1
WHERE frontline_requests_raw_v1.time >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
  AND frontline_requests_raw_v1.time < toUnixTimestamp64Milli(toDateTime64(toStartOfInterval(now(), INTERVAL 5 MINUTE), 3))
GROUP BY region;

CREATE MATERIALIZED VIEW anomaly_requests_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'requests' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM frontline_requests_raw_v1
GROUP BY region;

INSERT INTO anomaly_source_watermarks_v1
SELECT
  'resources' AS source,
  region,
  max(toStartOfMinute(fromUnixTimestamp64Milli(ts))) AS time
FROM instance_checkpoints_v1
WHERE ts >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
  AND ts < toUnixTimestamp64Milli(toDateTime64(toStartOfInterval(now(), INTERVAL 5 MINUTE), 3))
GROUP BY region;

CREATE MATERIALIZED VIEW anomaly_resources_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'resources' AS source,
  region,
  max(toStartOfMinute(fromUnixTimestamp64Milli(ts))) AS time
FROM instance_checkpoints_v1
GROUP BY region;

CREATE TABLE frontline_requests_anomaly_per_5m_v1 (
  time DateTime,
  workspace_id String,
  anomaly_shard UInt8 MATERIALIZED cityHash64(workspace_id) % 256,
  project_id String,
  app_id String,
  environment_id String,
  error_5xx SimpleAggregateFunction(sum, Int64),
  error_4xx SimpleAggregateFunction(sum, Int64),
  requests SimpleAggregateFunction(sum, Int64)
)
ENGINE = AggregatingMergeTree()
ORDER BY (anomaly_shard, workspace_id, project_id, app_id, environment_id, time)
PARTITION BY toYYYYMM(time)
TTL time + INTERVAL 30 DAY DELETE;

INSERT INTO frontline_requests_anomaly_per_5m_v1
SELECT
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sumIf(count, response_status >= 500 AND response_status < 600) AS error_5xx,
  sumIf(count, response_status >= 400 AND response_status < 500) AS error_4xx,
  sum(count) AS requests
FROM frontline_requests_per_5m_v1
WHERE time >= now() - INTERVAL 7 DAY
  AND time < toStartOfInterval(now(), INTERVAL 5 MINUTE)
  AND (workspace_id, project_id, app_id, environment_id, time) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, time
    FROM frontline_requests_anomaly_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id;

CREATE MATERIALIZED VIEW frontline_requests_anomaly_per_5m_mv_v1
TO frontline_requests_anomaly_per_5m_v1 AS
SELECT
  time,
  workspace_id,
  project_id,
  app_id,
  environment_id,
  sumIf(count, response_status >= 500 AND response_status < 600) AS error_5xx,
  sumIf(count, response_status >= 400 AND response_status < 500) AS error_4xx,
  sum(count) AS requests
FROM frontline_requests_per_5m_v1
GROUP BY time, workspace_id, project_id, app_id, environment_id;

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

INSERT INTO instance_resources_container_per_5m_v1
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
WHERE ts >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
  AND ts < toUnixTimestamp64Milli(toDateTime64(toStartOfInterval(now(), INTERVAL 5 MINUTE), 3))
  AND (workspace_id, project_id, app_id, environment_id, instance_id, container_uid, toStartOfInterval(fromUnixTimestamp64Milli(ts), INTERVAL 5 MINUTE)) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, instance_id, container_uid, time
    FROM instance_resources_container_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid;

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
