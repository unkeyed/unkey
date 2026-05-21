-- name: GlobalCountersListAll :many
-- GlobalCountersListAll returns raw per-region global counter rows for tests
-- that need to assert which region wrote which observation. Production callers
-- should use GlobalCountersImported instead so MySQL does the aggregation.
SELECT
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    region,
    count,
    expires_at,
    updated_at
FROM ratelimit_global_counters;
