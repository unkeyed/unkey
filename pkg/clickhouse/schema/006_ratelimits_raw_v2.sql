CREATE TABLE ratelimits_raw_v2 (
  -- the request id for correlation with traces and logs
  request_id String,
  -- unix milli
  time Int64 CODEC (Delta, LZ4),
  workspace_id String,
  namespace_id String,
  identifier String,
  -- whether the ratelimit check passed or was blocked
  passed Bool,
  -- Latency in milliseconds for this ratelimit check
  latency Float64,
  -- possible override id that was used for this check
  override_id String,
  -- what limit was checked against
  limit UInt64,
  -- how many remaining uses we allow
  remaining UInt64,
  -- when the limit will reset at (absolute unix milliseconds time)
  reset_at Int64 CODEC (Delta, LZ4),
  -- tokens charged against the limit on this decision (>=1; 0 for events
  -- recorded before token tracking shipped)
  tokens UInt64,
  -- Where the decision originated: 'api' (public API) or 'gateway' (Unkey
  -- Deploy's ratelimit policy). Billing rollups exclude 'gateway'.
  source LowCardinality (String) DEFAULT 'api',
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, namespace_id)
TTL toDateTime (fromUnixTimestamp64Milli (time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000;

