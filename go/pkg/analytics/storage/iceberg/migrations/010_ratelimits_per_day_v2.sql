-- Iceberg table for per-day aggregated ratelimit events
-- This mirrors ClickHouse schema/009_ratelimits_per_day_v2.sql
CREATE TABLE IF NOT EXISTS ratelimits_per_day_v2 (
  time TIMESTAMP NOT NULL,
  workspace_id STRING NOT NULL,
  namespace_id STRING NOT NULL,
  count BIGINT NOT NULL,
  latency_avg DOUBLE
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  'history.expire.max-snapshot-age-ms' = '2592000000'
);