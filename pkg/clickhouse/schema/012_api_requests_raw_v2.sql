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

