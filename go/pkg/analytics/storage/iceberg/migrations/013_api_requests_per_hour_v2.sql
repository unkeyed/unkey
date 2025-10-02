-- Iceberg table for per-hour aggregated API request events
-- This mirrors ClickHouse schema/014_api_requests_per_hour_v2.sql
CREATE TABLE IF NOT EXISTS api_requests_per_hour_v2 (
  time TIMESTAMP NOT NULL,
  workspace_id STRING NOT NULL,
  host STRING NOT NULL,
  count BIGINT NOT NULL
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'history.expire.max-snapshot-age-ms' = '2592000000'
);