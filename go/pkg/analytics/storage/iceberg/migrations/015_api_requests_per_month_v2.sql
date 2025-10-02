-- Iceberg table for per-month aggregated API request events
-- This mirrors ClickHouse schema/016_api_requests_per_month_v2.sql
CREATE TABLE IF NOT EXISTS api_requests_per_month_v2 (
  time TIMESTAMP NOT NULL,
  workspace_id STRING NOT NULL,
  host STRING NOT NULL,
  count BIGINT NOT NULL
)
USING iceberg
PARTITIONED BY (months(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'history.expire.max-snapshot-age-ms' = '2592000000'
);