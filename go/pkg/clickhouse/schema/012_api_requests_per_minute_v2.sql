CREATE TABLE api_requests_per_minute_v2 (
  time DateTime,
  workspace_id String,
  path String,
  response_status Int,
  host String,
  -- Upper case HTTP method
  -- Examples: "GET", "POST", "PUT", "DELETE"
  method LowCardinality (String),
  count Int64
) ENGINE = SummingMergeTree ()
ORDER BY
  (
    workspace_id,
    time,
    host,
    path,
    response_status,
    method
  );

CREATE MATERIALIZED VIEW api_requests_per_minute_mv_v2 TO api_requests_per_minute_v2 AS
SELECT
  workspace_id,
  path,
  response_status,
  host,
  method,
  count(*) as count,
  toStartOfMinute (fromUnixTimestamp64Milli (time)) AS time
FROM
  api_requests_raw_v2
GROUP BY
  workspace_id,
  path,
  response_status,
  host,
  method,
  time;
