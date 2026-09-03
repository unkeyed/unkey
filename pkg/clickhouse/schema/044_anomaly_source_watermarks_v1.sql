-- A two-row rollup avoids scanning fleet tables whose primary key starts with
-- workspace_id. A time predicate alone cannot skip their 24-hour data parts.
CREATE TABLE anomaly_source_watermarks_v1 (
  source LowCardinality(String),
  time SimpleAggregateFunction(max, DateTime)
)
ENGINE = AggregatingMergeTree()
ORDER BY source;

CREATE MATERIALIZED VIEW anomaly_requests_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'requests' AS source,
  max(time) AS time
FROM frontline_requests_per_5m_v1;

CREATE MATERIALIZED VIEW anomaly_resources_watermark_mv_v1
TO anomaly_source_watermarks_v1 AS
SELECT
  'resources' AS source,
  max(time) AS time
FROM instance_resources_per_minute_v1;
