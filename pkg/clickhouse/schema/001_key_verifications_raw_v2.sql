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
  external_id String,
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

  -- Per-row TTL stamp (unix milli). DEFAULT preserves the table's
  -- historical 90 day retention; once writers populate this from the
  -- workspace's logs_retention_days quota, retention becomes per-plan.
  expires_at Int64 DEFAULT time + 7776000000 CODEC(Delta, LZ4),

  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identity_id (identity_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_external_id (external_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_key_id (key_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_tags (tags) TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, time, key_space_id, outcome)
TTL toDateTime(fromUnixTimestamp64Milli(expires_at)) DELETE
SETTINGS non_replicated_deduplication_window = 10000
;

