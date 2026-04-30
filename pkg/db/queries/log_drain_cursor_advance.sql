-- name: AdvanceLogDrainCursor :execrows
-- Optimistic-lock cursor advance. Only the row that observed the previous
-- watermark wins; if a concurrent replica already moved the cursor, this
-- returns 0 rows affected and the caller replays the same window on the
-- next tick (idempotency keys take care of the dup at the provider).
UPDATE log_drain_cursors
SET inserted_at_ms = sqlc.arg(new_inserted_at_ms),
    fingerprint = sqlc.arg(new_fingerprint),
    updated_at = sqlc.arg(updated_at)
WHERE group_key = sqlc.arg(group_key)
  AND inserted_at_ms = sqlc.arg(prev_inserted_at_ms)
  AND fingerprint = sqlc.arg(prev_fingerprint);
