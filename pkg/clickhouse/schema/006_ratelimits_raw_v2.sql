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
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, namespace_id)
TTL toDateTime (fromUnixTimestamp64Milli (time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000;

-- Temporary materialized view to sync new writes from v1 to v2 during migration
-- This ensures zero-downtime migration by duplicating all new inserts
-- DROP this view after migration is complete and application switches to v2
CREATE MATERIALIZED VIEW temp_sync_ratelimits_raw_v1_to_v2 TO ratelimits_raw_v2 AS
SELECT
  request_id,
  time,
  workspace_id,
  namespace_id,
  identifier,
  passed,
   -- v1 doesn't have any of those columns
  0.0 as latency,
  '' as override_id,
  0 as `limit`,
  0 as remaining,
  0 as reset_at
FROM
  ratelimits.raw_ratelimits_v1;
