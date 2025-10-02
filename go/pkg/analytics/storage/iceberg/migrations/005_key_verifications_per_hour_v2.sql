-- Iceberg table for per-hour aggregated key verification events
-- This mirrors ClickHouse schema/003_key_verifications_per_hour_v2.sql
CREATE TABLE IF NOT EXISTS key_verifications_per_hour_v2 (
  -- Rounded to start of hour
  time TIMESTAMP NOT NULL,

  workspace_id STRING NOT NULL,
  key_space_id STRING NOT NULL,
  outcome STRING NOT NULL,
  tags ARRAY<STRING>,

  -- Aggregated metrics
  count BIGINT NOT NULL,
  spent_credits BIGINT,
  latency_avg DOUBLE,
  latency_p75 DOUBLE,
  latency_p99 DOUBLE
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: Will be set based on workspace config (default 1 month)
  'history.expire.max-snapshot-age-ms' = '2592000000'
);