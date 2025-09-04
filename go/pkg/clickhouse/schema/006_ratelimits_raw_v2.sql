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
  INDEX idx_request_id (request_id) TYPE minmax GRANULARITY 1
) ENGINE = MergeTree ()
PARTITION BY
  toYYYYMM (fromUnixTimestamp64Milli (time))
ORDER BY
  (workspace_id, time, namespace_id, identifier) TTL toDateTime (fromUnixTimestamp64Milli (time)) + INTERVAL 1 MONTH DELETE SETTINGS non_replicated_deduplication_window = 10000;

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
  0.0 as latency -- v1 doesn't have this column, default to 0.0
FROM
  ratelimits.raw_ratelimits_v1;
