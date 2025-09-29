CREATE TABLE api_requests_per_month_v2 (
  time Date,
  workspace_id String,
  path String,
  response_status Int,
  host String,
  -- Upper case HTTP method
  -- Examples: "GET", "POST", "PUT", "DELETE"
  method LowCardinality (String),
  count SimpleAggregateFunction(sum, Int64),
  INDEX idx_host (host) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_path (path) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_method (method) TYPE bloom_filter GRANULARITY 1
) ENGINE = SummingMergeTree ()
ORDER BY
  (
    workspace_id,
    time,
    response_status,
    host,
    method,
    path
  )
  TTL time + INTERVAL 3 YEAR DELETE;

CREATE MATERIALIZED VIEW api_requests_per_month_mv_v2 TO api_requests_per_month_v2 AS
SELECT
  workspace_id,
  path,
  response_status,
  host,
  method,
  sum(count) as count,
  toDate(toStartOfMonth(time)) AS time
FROM
  api_requests_per_day_v2
GROUP BY
  workspace_id,
  path,
  response_status,
  host,
  method,
  time;
