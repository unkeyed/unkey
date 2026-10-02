CREATE TABLE ratelimits_raw_v2 (
  -- the request id for correlation with traces and logs
  request_id String,
  -- unix milli
  time Int64 CODEC (Delta, LZ4),
  inserted_at Int64 DEFAULT toUnixTimestamp64Milli(now64(3)) CODEC(Delta, ZSTD(1)),
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
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1,
  PROJECTION proj_logdrain
  (
    SELECT workspace_id, inserted_at, request_id, _part_offset
    ORDER BY workspace_id, inserted_at, request_id
  )
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, namespace_id)
TTL toDateTime (fromUnixTimestamp64Milli (time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000,
  allow_part_offset_column_in_projections = 1,
  deduplicate_merge_projection_mode = 'rebuild';
