-- name: UpsertRatelimitGlobalCounters :exec
-- UpsertRatelimitGlobalCounters records one region's latest observation for a
-- sliding-window cell. The generated bulk variant is the hot path: conflicts
-- use GREATEST so concurrent writers for the same region collapse onto the
-- highest count, which keeps regional observations monotonic within a sequence.
INSERT INTO ratelimit_global_counters (
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    region,
    count,
    expires_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(namespace),
    sqlc.arg(identifier),
    sqlc.arg(duration_ms),
    sqlc.arg(sequence),
    sqlc.arg(region),
    sqlc.arg(count),
    sqlc.arg(expires_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    count = GREATEST(count, VALUES(count)),
    updated_at = VALUES(updated_at);
