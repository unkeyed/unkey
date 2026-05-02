-- name: AdvanceLogDrainCursor :execrows
-- Optimistic-lock advance on a per-drain cursor row. Only the row that
-- observed the previous (time_ms, last_id) wins; if a concurrent replica
-- already moved this drain's cursor, this returns 0 rows affected and
-- the caller treats the batch as having been delivered by the winner —
-- next tick replays from the new cursor.
UPDATE log_drain_cursors
SET time_ms = sqlc.arg(new_time_ms),
    last_id = sqlc.arg(new_last_id),
    updated_at = sqlc.arg(updated_at)
WHERE drain_id = sqlc.arg(drain_id)
  AND group_key = sqlc.arg(group_key)
  AND time_ms = sqlc.arg(prev_time_ms)
  AND last_id = sqlc.arg(prev_last_id);
