-- Iceberg table for API request events
-- This mirrors the ClickHouse schema in pkg/clickhouse/schema/012_api_requests_raw_v2.sql
CREATE TABLE IF NOT EXISTS api_requests_raw_v2 (
  request_id STRING NOT NULL,

  -- Unix timestamp in milliseconds
  time BIGINT NOT NULL,

  workspace_id STRING NOT NULL,
  host STRING NOT NULL,

  -- Upper case HTTP method: "GET", "POST", "PUT", "DELETE", etc.
  method STRING NOT NULL,

  path STRING NOT NULL,

  -- Raw query string (e.g., "a=b&c=d")
  query_string STRING,

  -- Parsed query parameters as map for efficient querying
  -- Example: {"a": ["b"], "c": ["d", "e"]} for multi-value params
  query_params MAP<STRING, ARRAY<STRING>>,

  -- Request headers as "Key: Value" pairs
  request_headers ARRAY<STRING>,

  request_body STRING,

  response_status INT,

  -- Response headers as "Key: Value" pairs
  response_headers ARRAY<STRING>,

  response_body STRING,

  -- Internal error message, empty if no error
  error STRING,

  -- Service latency in milliseconds
  service_latency BIGINT,

  user_agent STRING,
  ip_address STRING,
  region STRING
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: 1 month
  'history.expire.max-snapshot-age-ms' = '2592000000'
);