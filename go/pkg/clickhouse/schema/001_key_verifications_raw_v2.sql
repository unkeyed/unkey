CREATE TABLE key_verifications_raw_v2
(
-- the api request id, so we can correlate the verification with traces and logs
  request_id String,

  -- unix milli
  time Int64 CODEC(Delta, LZ4),

  workspace_id String,
  key_space_id String,
  -- Empty string if the key has no identity
  identity_id String,
  key_id String,

  -- Right now this is a 3 character airport code, but when we move to aws,
  -- this will be the region code such as `us-east-1`
  region LowCardinality(String),

  -- Examples:
  -- - "VALID"
  -- - "RATE_LIMITED"
  -- - "EXPIRED"
  -- - "DISABLED
  outcome LowCardinality(String),

  tags Array(String) DEFAULT [],

  -- The number of credits spent on this verification
  -- 0 means no credits were spent
  spent_credits Int64,

  -- Latency in milliseconds for this verification
  latency Float64,

  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identity_id (identity_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_key_id (key_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_tags (tags) TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, time, key_space_id, outcome)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000
;

-- Temporary materialized view to sync new writes from v1 to v2 during migration
-- This ensures zero-downtime migration by duplicating all new inserts
-- DROP this view after migration is complete and application switches to v2

CREATE MATERIALIZED VIEW  temp_sync_key_verifications_v1_to_v2
TO key_verifications_raw_v2
AS
SELECT
    request_id,
    time,
    workspace_id,
    key_space_id,
    identity_id,
    key_id,
    region,
    outcome,
    tags,
    0 as spent_credits,    -- v1 doesn't have this column, default to 0
    0.0 as latency         -- v1 doesn't have this column, default to 0.0
FROM verifications.raw_key_verifications_v1;
