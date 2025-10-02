-- Iceberg table for per-month aggregated key verification events
-- This mirrors ClickHouse schema/005_key_verifications_per_month_v2.sql
CREATE TABLE IF NOT EXISTS key_verifications_per_month_v2 (
  -- Rounded to start of month
  time TIMESTAMP NOT NULL,

  workspace_id STRING NOT NULL,
  key_space_id STRING NOT NULL,
  outcome STRING NOT NULL,

  -- Aggregated metrics
  count BIGINT NOT NULL,
  spent_credits BIGINT,
  latency_avg DOUBLE,
  latency_p75 DOUBLE,
  latency_p99 DOUBLE
)
USING iceberg
PARTITIONED BY (months(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: Will be set based on workspace config (default 1 month)
  'history.expire.max-snapshot-age-ms' = '2592000000'
);