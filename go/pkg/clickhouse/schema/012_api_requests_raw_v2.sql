CREATE TABLE api_requests_raw_v2 (
  request_id String,
  -- unix milli
  time Int64 CODEC(Delta, LZ4),
  workspace_id String,
  host String,
  -- Upper case HTTP method
  -- Examples: "GET", "POST", "PUT", "DELETE"
  method LowCardinality (String),
  path String,
  -- Raw query string (e.g., "a=b&c=d")
  query_string String,
  -- Parsed query parameters as map for efficient querying
  -- Example: {"a": ["b"], "c": ["d", "e"]} for multi-value params
  query_params Map(String, Array(String)),
  -- "Key: Value" pairs
  request_headers Array(String),
  request_body String,
  response_status Int,
  -- "Key: Value" pairs
  response_headers Array(String),
  response_body String,
  -- internal err.Error() string, empty if no error
  error String,
  -- milliseconds
  service_latency Int64,
  user_agent String,
  ip_address String,
  region LowCardinality (String),
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, request_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000;

-- Temporary materialized view to sync new writes from v1 to v2 during migration
-- This ensures zero-downtime migration by duplicating all new inserts
-- DROP this view after migration is complete and application switches to v2
CREATE MATERIALIZED VIEW temp_sync_metrics_v1_to_v2 TO api_requests_raw_v2 AS
SELECT
  request_id,
  time,
  workspace_id,
  host,
  method,
  path,
  '' as query_string,
  CAST(mapFromArrays(CAST([],'Array(String)'), CAST([],'Array(Array(String))')), 'Map(String, Array(String))') AS query_params,
  request_headers,
  request_body,
  response_status,
  response_headers,
  response_body,
  error,
  service_latency,
  user_agent,
  ip_address,
  '' as region
FROM
  metrics.raw_api_requests_v1;
