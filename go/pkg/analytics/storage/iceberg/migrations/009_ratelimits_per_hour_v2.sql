-- Iceberg table for per-hour aggregated ratelimit events
-- This mirrors ClickHouse schema/008_ratelimits_per_hour_v2.sql
CREATE TABLE IF NOT EXISTS ratelimits_per_hour_v2 (
  time TIMESTAMP NOT NULL,
  workspace_id STRING NOT NULL,
  namespace_id STRING NOT NULL,
  passed BOOLEAN NOT NULL,
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
  'history.expire.max-snapshot-age-ms' = '2592000000'
);