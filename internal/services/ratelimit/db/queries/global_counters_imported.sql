-- name: GlobalCountersImported :many
-- GlobalCountersImported returns the caller's own-region count and the sum of
-- foreign-region contributions for every still-active window cell. Receivers
-- fold the own-region count into counterEntry.val and the foreign-region count
-- into counterEntry.globalCount; keeping them separate prevents own traffic
-- from being double-counted as imported global state. Aggregation runs in MySQL
-- because the application only ever uses these sums, so transferring per-region
-- rows just to collapse them in Go wastes bandwidth and memory. The sums are
-- cast to SIGNED so sqlc maps them to int64, matching atomic.Int64 in the caller.
SELECT
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    CAST(SUM(CASE WHEN region = sqlc.arg("self_region") THEN count ELSE 0 END) AS SIGNED) AS regional,
    CAST(SUM(CASE WHEN region != sqlc.arg("self_region") THEN count ELSE 0 END) AS SIGNED) AS imported
FROM ratelimit_global_counters
WHERE expires_at > sqlc.arg("now")
GROUP BY workspace_id, namespace, identifier, duration_ms, sequence;
