-- Ratelimit raw events v3: adds end-user identity attribution.
-- identity_id/external_id are resolved at check time by matching the
-- caller-supplied identifier against the workspace's identities (the
-- documented convention is identifier == the end-user's external_id).
-- Bare identifiers that match no identity write empty strings and stay
-- unattributed. Ingestion writes ONLY to this table; the mirror MV below
-- forwards rows into ratelimits_raw_v2 so the existing v2 rollup cascade
-- and dashboards keep working unchanged.
CREATE TABLE ratelimits_raw_v3 (
  -- the request id for correlation with traces and logs
  request_id String,
  -- unix milli
  time Int64 CODEC (Delta, LZ4),
  workspace_id String,
  namespace_id String,
  identifier String,
  -- resolved end-user identity, empty when the identifier matched none
  identity_id String,
  external_id String,
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
  -- tokens charged against the limit on this decision
  tokens UInt64,
  INDEX idx_request_id (request_id) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_identifier (identifier) TYPE bloom_filter GRANULARITY 1,
  INDEX idx_external_id (external_id) TYPE bloom_filter GRANULARITY 1
) ENGINE = MergeTree ()
ORDER BY
  (workspace_id, time, namespace_id)
TTL toDateTime (fromUnixTimestamp64Milli (time)) + INTERVAL 1 MONTH DELETE
SETTINGS non_replicated_deduplication_window = 10000;

CREATE MATERIALIZED VIEW ratelimits_raw_v2_mirror_mv_v1 TO ratelimits_raw_v2 AS
SELECT
  request_id,
  time,
  workspace_id,
  namespace_id,
  identifier,
  passed,
  latency,
  override_id,
  `limit`,
  remaining,
  reset_at,
  tokens
FROM
  ratelimits_raw_v3;
