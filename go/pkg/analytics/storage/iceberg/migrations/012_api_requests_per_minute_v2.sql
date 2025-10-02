-- Iceberg table for per-minute aggregated API request events
-- This mirrors ClickHouse schema/013_api_requests_per_minute_v2.sql
CREATE TABLE IF NOT EXISTS api_requests_per_minute_v2 (
  time TIMESTAMP NOT NULL,
  workspace_id STRING NOT NULL,
  host STRING NOT NULL,
  method STRING NOT NULL,
  path STRING NOT NULL,
  response_status INT NOT NULL,
  count BIGINT NOT NULL
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'history.expire.max-snapshot-age-ms' = '604800000'
);