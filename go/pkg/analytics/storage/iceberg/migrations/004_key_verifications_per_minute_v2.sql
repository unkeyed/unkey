-- Iceberg table for per-minute aggregated key verification events
-- This mirrors ClickHouse schema/002_key_verifications_per_minute_v2.sql
-- Note: Unlike ClickHouse materialized views, aggregation must be done externally
--       (via Spark/Flink jobs reading from raw table and writing here)
CREATE TABLE IF NOT EXISTS key_verifications_per_minute_v2 (
  -- Rounded to start of minute
  time TIMESTAMP NOT NULL,

  workspace_id STRING NOT NULL,
  key_space_id STRING NOT NULL,
  identity_id STRING,
  key_id STRING NOT NULL,
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
  -- Retention: 7 days (shorter than raw data since this is aggregated)
  -- Will be overridden by workspace-specific retention config
  'history.expire.max-snapshot-age-ms' = '604800000'
);