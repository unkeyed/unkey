-- Anomaly detection reads this app-level rollup every 5 minutes. Reading the
-- per-minute dashboard table for a 24-hour baseline exceeded 14 GiB at 100,000
-- one-instance apps and could not complete on a 32 GiB worker.
--
-- The source is dashboards-only and can double-count rare retry inserts. That
-- tradeoff is acceptable for anomaly detection, but this table must not be
-- used for billing. Memory uses a separate container-grain 5-minute rollup to
-- preserve equal weighting across instances.
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

-- Seven days matches the dashboard overview horizon while bounding migration
-- cost. Aggregate sum states do not deduplicate replayed inserts, so excluding
-- existing rollup keys makes retries fill missing keys without doubling them.
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

-- A per-region rollup avoids scanning fleet tables whose primary key starts
-- with workspace_id. The detector uses the least watermark among regions with
-- ingest in the last two hours, so a healthy region cannot hide a lagging one.
CREATE TABLE anomaly_source_watermarks_v1 (
  source LowCardinality(String),
  region LowCardinality(String),
  time SimpleAggregateFunction(max, DateTime)
)
ENGINE = AggregatingMergeTree()
ORDER BY (source, region);

CREATE MATERIALIZED VIEW anomaly_requests_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'requests' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM frontline_requests_raw_v1
GROUP BY region;

-- The bounded seven-day replay gives every active region an immediate
-- watermark. Replaying max aggregate states is idempotent under merges.
INSERT INTO anomaly_source_watermarks_v1
SELECT
  'requests' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM frontline_requests_raw_v1
WHERE frontline_requests_raw_v1.time >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
GROUP BY region;

CREATE MATERIALIZED VIEW anomaly_resources_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'resources' AS source,
  region,
  max(toStartOfMinute(fromUnixTimestamp64Milli(ts))) AS time
FROM instance_checkpoints_v1
GROUP BY region;

INSERT INTO anomaly_source_watermarks_v1
SELECT
  'resources' AS source,
  region,
  max(toStartOfMinute(fromUnixTimestamp64Milli(ts))) AS time
FROM instance_checkpoints_v1
WHERE ts >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
GROUP BY region;

CREATE MATERIALIZED VIEW anomaly_instance_events_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'instance_events' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM instance_events_raw_v1
GROUP BY region;

INSERT INTO anomaly_source_watermarks_v1
SELECT
  'instance_events' AS source,
  region,
  max(toStartOfInterval(fromUnixTimestamp64Milli(time), INTERVAL 5 MINUTE)) AS time
FROM instance_events_raw_v1
WHERE instance_events_raw_v1.time >= toUnixTimestamp64Milli(toDateTime64(now() - INTERVAL 7 DAY, 3))
GROUP BY region;

-- App-level request counts avoid rebuilding groups x status-code buckets in
-- every anomaly shard. This table is dashboards-only because chained MVs can
-- double-count retried inserts before source-table deduplication.
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

-- Seven days matches the dashboard overview horizon while bounding migration
-- cost. Aggregate sum states do not deduplicate replayed inserts, so excluding
-- existing rollup keys makes retries fill missing keys without doubling them.
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
  AND (workspace_id, project_id, app_id, environment_id, time) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, time
    FROM frontline_requests_anomaly_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id;

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

-- Seven days matches the dashboard overview horizon while bounding migration
-- cost. Aggregate sum states do not deduplicate replayed inserts, so excluding
-- existing rollup keys makes retries fill missing keys without doubling them.
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
  AND (workspace_id, project_id, app_id, environment_id, instance_id, container_uid, toStartOfInterval(instance_resources_per_minute_v1.time, INTERVAL 5 MINUTE)) NOT IN (
    SELECT workspace_id, project_id, app_id, environment_id, instance_id, container_uid, time
    FROM instance_memory_container_per_5m_v1
    WHERE time >= now() - INTERVAL 7 DAY
  )
GROUP BY time, workspace_id, project_id, app_id, environment_id, instance_id, container_uid;
