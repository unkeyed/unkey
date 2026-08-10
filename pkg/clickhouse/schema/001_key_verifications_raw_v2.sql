CREATE TABLE key_verifications_raw_v2
(
-- the api request id, so we can correlate the verification with traces and logs
  request_id String CODEC(ZSTD(1)),

  -- unix milli
  time Int64 CODEC(Delta, ZSTD(1)),

  workspace_id String CODEC(ZSTD(1)),
  key_space_id String CODEC(ZSTD(1)),
  -- Empty string if the key has no identity
  identity_id String CODEC(ZSTD(1)),
  external_id String CODEC(ZSTD(1)),
  key_id String CODEC(ZSTD(1)),

  -- Right now this is a 3 character airport code, but when we move to aws,
  -- this will be the region code such as `us-east-1`
  region LowCardinality(String) CODEC(ZSTD(1)),

  -- Where the verification originated: 'api' (public API) or 'gateway'
  -- (an Unkey Gateway key-auth policy). Billing rollups exclude 'gateway'.
  source LowCardinality(String) DEFAULT 'api' CODEC(ZSTD(1)),

  -- The Unkey Gateway app that ran the verification. Empty for verifications
  -- that did not originate from Gateway.
  app_id LowCardinality(String) DEFAULT '' CODEC(ZSTD(1)),

  -- Examples:
  -- - "VALID"
  -- - "RATE_LIMITED"
  -- - "EXPIRED"
  -- - "DISABLED
  outcome LowCardinality(String) CODEC(ZSTD(1)),

  tags Array(String) DEFAULT [] CODEC(ZSTD(1)),

  -- The number of credits spent on this verification
  -- 0 means no credits were spent
  spent_credits Int64 CODEC(ZSTD(1)),

  -- Latency in milliseconds for this verification
  latency Float64 CODEC(ZSTD(1)),

  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identity_id (identity_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_external_id (external_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_key_id (key_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_tags (tags) TYPE bloom_filter GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (workspace_id, time, key_space_id, outcome)
TTL toDateTime(fromUnixTimestamp64Milli(time)) + INTERVAL 90 DAY DELETE
SETTINGS non_replicated_deduplication_window = 10000
;
