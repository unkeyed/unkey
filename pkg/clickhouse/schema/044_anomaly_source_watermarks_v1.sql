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
