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

-- name: GlobalCountersImportedSince :many
-- GlobalCountersImportedSince first identifies logical window cells with an
-- active region row updated inside the overlapped watermark, then aggregates
-- every active region row for those cells. Filtering only the changed physical
-- rows would omit unchanged regions and undercount the imported total.
SELECT
    counters.workspace_id,
    counters.namespace,
    counters.identifier,
    counters.duration_ms,
    counters.sequence,
    CAST(SUM(CASE WHEN counters.region = sqlc.arg("self_region") THEN counters.count ELSE 0 END) AS SIGNED) AS regional,
    CAST(SUM(CASE WHEN counters.region != sqlc.arg("self_region") THEN counters.count ELSE 0 END) AS SIGNED) AS imported
FROM ratelimit_global_counters AS counters
INNER JOIN (
    SELECT DISTINCT
        recent.workspace_id,
        recent.namespace,
        recent.identifier,
        recent.duration_ms,
        recent.sequence
    FROM ratelimit_global_counters AS recent
    WHERE recent.expires_at > sqlc.arg("now")
      AND recent.updated_at >= sqlc.arg("updated_after")
) AS changed
    ON changed.workspace_id = counters.workspace_id
    AND changed.namespace = counters.namespace
    AND changed.identifier = counters.identifier
    AND changed.duration_ms = counters.duration_ms
    AND changed.sequence = counters.sequence
WHERE counters.expires_at > sqlc.arg("now")
GROUP BY counters.workspace_id, counters.namespace, counters.identifier, counters.duration_ms, counters.sequence;
