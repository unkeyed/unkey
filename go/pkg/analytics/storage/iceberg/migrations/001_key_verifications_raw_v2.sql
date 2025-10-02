-- Iceberg table for key verification events
-- This mirrors the ClickHouse schema in pkg/clickhouse/schema/001_key_verifications_raw_v2.sql
CREATE TABLE IF NOT EXISTS key_verifications_raw_v2 (
  -- The API request ID for correlating verification with traces and logs
  request_id STRING NOT NULL,

  -- Unix timestamp in milliseconds
  time BIGINT NOT NULL,

  workspace_id STRING NOT NULL,
  key_space_id STRING NOT NULL,

  -- Empty string if the key has no identity
  identity_id STRING,
  key_id STRING NOT NULL,

  -- Region code (3 character airport code or AWS region like 'us-east-1')
  region STRING,

  -- Verification outcome: "VALID", "RATE_LIMITED", "EXPIRED", "DISABLED", etc.
  outcome STRING NOT NULL,

  -- Tags associated with the key
  tags ARRAY<STRING>,

  -- Number of credits spent on this verification (0 means no credits were spent)
  spent_credits BIGINT,

  -- Latency in milliseconds for this verification
  latency DOUBLE
)
USING iceberg
PARTITIONED BY (days(time), workspace_id)
TBLPROPERTIES (
  'write.format.default' = 'parquet',
  'write.parquet.compression-codec' = 'zstd',
  -- Retention: 1 month (Iceberg uses time-travel, configure externally)
  'history.expire.max-snapshot-age-ms' = '2592000000'
);