-- name: GetLogDrainCursor :one
SELECT group_key, inserted_at_ms, fingerprint, updated_at
FROM log_drain_cursors
WHERE group_key = ?;
