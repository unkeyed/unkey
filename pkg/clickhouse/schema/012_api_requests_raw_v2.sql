CREATE TABLE api_requests_raw_v2 (
  request_id String CODEC(ZSTD(3)),
  -- unix milli
  time Int64 CODEC(Delta, ZSTD(3)),
  -- When ClickHouse accepted the row (unix milli). Writers omit it so the
  -- server clock stamps every row, which gives log drains a cursor that
  -- cannot lag behind buffered or retried inserts. Rows written before the
  -- column existed hold 0.
  inserted_at Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(3)),
  workspace_id String CODEC(ZSTD(3)),
  host String CODEC(ZSTD(3)),
  -- Upper case HTTP method
  -- Examples: "GET", "POST", "PUT", "DELETE"
  method LowCardinality (String) CODEC(ZSTD(3)),
  path String CODEC(ZSTD(3)),
  -- Raw query string (e.g., "a=b&c=d")
  query_string String CODEC(ZSTD(3)),
  -- Parsed query parameters as map for efficient querying
  -- Example: {"a": ["b"], "c": ["d", "e"]} for multi-value params
  query_params Map(String, Array(String)) CODEC(ZSTD(3)),
  -- "Key: Value" pairs
  request_headers Array(String) CODEC(ZSTD(3)),
  request_body String CODEC(ZSTD(3)),
  response_status Int,
  -- "Key: Value" pairs
  response_headers Array(String) CODEC(ZSTD(3)),
  response_body String CODEC(ZSTD(3)),
  -- internal err.Error() string, empty if no error
  error String CODEC(ZSTD(3)),
  -- milliseconds
  service_latency Int64,
  user_agent String CODEC(ZSTD(3)),
  ip_address String CODEC(ZSTD(3)),
  region LowCardinality (String) CODEC(ZSTD(3)),
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, request_id)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000;

