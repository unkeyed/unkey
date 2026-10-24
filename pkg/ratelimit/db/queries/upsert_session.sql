-- name: UpsertSession :exec
-- UpsertSession writes one session's current values. GREATEST keeps the stored
-- consumed_ns when it is larger, so a retried or out-of-order write cannot undo
-- a newer one. anchor_at_ns and compact_after_ms are set when the row is
-- created and never change.
INSERT INTO ratelimit_state (
    fingerprint,
    session_id,
    anchor_at_ns,
    consumed_ns,
    compact_after_ms
) VALUES (
    sqlc.arg(fingerprint),
    sqlc.arg(session_id),
    sqlc.arg(anchor_at_ns),
    sqlc.arg(consumed_ns),
    sqlc.arg(compact_after_ms)
)
ON DUPLICATE KEY UPDATE
    consumed_ns = GREATEST(consumed_ns, VALUES(consumed_ns));
