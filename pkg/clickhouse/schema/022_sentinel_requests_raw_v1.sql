CREATE TABLE sentinel_requests_raw_v1 (
  request_id String,
  -- unix milli
  time Int64 CODEC(Delta, LZ4),
  workspace_id String,
  environment_id String,
  project_id String,
  sentinel_id String,
  deployment_id String,
  instance_id String,
  instance_address String,
  region LowCardinality (String),
  -- Upper case HTTP method
  method LowCardinality (String),
  host String,
  path String,
  -- Raw query string
  query_string String,
  -- Parsed query parameters
  query_params Map(String, Array(String)),
  -- "Key: Value" pairs
  request_headers Array(String),
  request_body String,
  response_status Int32,
  -- "Key: Value" pairs
  response_headers Array(String),
  response_body String,
  -- Error message if any
  error String,
  user_agent String,
  ip_address String,
  -- Milliseconds - total latency
  service_latency Int64,
  -- Milliseconds - instance processing time
  instance_latency Int64,
  -- Milliseconds - sentinel overhead
  sentinel_processing_latency Int64,
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_deployment_id (deployment_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_instance_id (instance_id) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree()
ORDER BY
  (workspace_id, project_id, deployment_id, time)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 30 DAY DELETE
SETTINGS non_replicated_deduplication_window = 10000;
