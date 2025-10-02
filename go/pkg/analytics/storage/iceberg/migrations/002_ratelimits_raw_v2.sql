-- Iceberg table for ratelimit events
-- This mirrors the ClickHouse schema in pkg/clickhouse/schema/006_ratelimits_raw_v2.sql
CREATE TABLE IF NOT EXISTS ratelimits_raw_v2 (
  -- The request ID for correlation with traces and logs
  request_id STRING NOT NULL,

  -- Unix timestamp in milliseconds
  time BIGINT NOT NULL,

  workspace_id STRING NOT NULL,
  namespace_id STRING NOT NULL,
  identifier STRING NOT NULL,

  -- Whether the ratelimit check passed or was blocked
  passed BOOLEAN NOT NULL,

  -- Latency in milliseconds for this ratelimit check
  latency DOUBLE,

  -- Possible override ID that was used for this check
  override_id STRING,

  -- What limit was checked against
  limit BIGINT,

  -- How many remaining uses we allow
  remaining BIGINT,

  -- When the limit will reset (absolute unix milliseconds time)
  reset_at BIGINT
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: 1 month
  'history.expire.max-snapshot-age-ms' = '2592000000'
);