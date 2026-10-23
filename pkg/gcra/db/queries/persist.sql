-- name: Persist :exec
-- Persist upserts one writer's row. spent_ns and fresh_start_ns only grow, so
-- GREATEST makes a retried or reordered write harmless without revisions.
-- updated_at_ms is assigned first because MySQL evaluates the assignments left
-- to right, and it must compare against the old values.
INSERT INTO ratelimit_gcra (
    name,
    writer_id,
    spent_ns,
    fresh_start_ns,
    expires_at_ns,
    updated_at_ms
) VALUES (
    sqlc.arg(name),
    sqlc.arg(writer_id),
    sqlc.arg(spent_ns),
    sqlc.arg(fresh_start_ns),
    sqlc.arg(expires_at_ns),
    CAST(UNIX_TIMESTAMP(NOW(3)) * 1000 AS SIGNED)
)
ON DUPLICATE KEY UPDATE
    updated_at_ms = IF(VALUES(spent_ns) > spent_ns OR VALUES(fresh_start_ns) > fresh_start_ns, VALUES(updated_at_ms), updated_at_ms),
    spent_ns = GREATEST(spent_ns, VALUES(spent_ns)),
    fresh_start_ns = GREATEST(fresh_start_ns, VALUES(fresh_start_ns)),
    expires_at_ns = GREATEST(expires_at_ns, VALUES(expires_at_ns));
