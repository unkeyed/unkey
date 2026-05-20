-- name: GlobalCountersImported :many
-- GlobalCountersImported returns the sum of foreign-region contributions for
-- every still-active window cell, with each region's row excluded from its
-- own caller. Receivers fold the returned `imported` directly into
-- counterEntry.imported via atomicMax; aggregation runs in MySQL because
-- the application only ever uses the sum, so transferring per-region rows
-- just to collapse them in Go wastes bandwidth and memory. The SUM is cast
-- to SIGNED so sqlc maps it to int64, matching atomic.Int64 in the caller.
SELECT
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    CAST(SUM(count) AS SIGNED) AS imported
FROM ratelimit_global_counters
WHERE expires_at > sqlc.arg("now")
  AND region != sqlc.arg("self_region")
GROUP BY workspace_id, namespace, identifier, duration_ms, sequence;
