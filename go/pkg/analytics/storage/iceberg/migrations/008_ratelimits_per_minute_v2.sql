-- Iceberg table for per-minute aggregated ratelimit events
-- This mirrors ClickHouse schema/007_ratelimits_per_minute_v2.sql
CREATE TABLE IF NOT EXISTS ratelimits_per_minute_v2 (
  -- Rounded to start of minute
  time TIMESTAMP NOT NULL,

  workspace_id STRING NOT NULL,
  namespace_id STRING NOT NULL,
  identifier STRING NOT NULL,
  passed BOOLEAN NOT NULL,

  -- Aggregated metrics
  count BIGINT NOT NULL,
  latency_avg DOUBLE,
  latency_p75 DOUBLE,
  latency_p99 DOUBLE
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: 7 days (shorter than raw data)
  'history.expire.max-snapshot-age-ms' = '604800000'
);