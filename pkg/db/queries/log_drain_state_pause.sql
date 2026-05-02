-- name: PauseLogDrain :exec
-- Sets paused_reason on a drain after consecutive_failures crosses the
-- coordinator's PauseAfterFailures threshold. ListEnabledLogDrains filters
-- on `paused_reason IS NULL OR paused_reason = ''`, so this single column
-- write is enough to take the drain out of rotation on the next tick.
-- Resume is a dashboard action that clears paused_reason back to NULL.
UPDATE log_drain_state
SET paused_reason = sqlc.arg(paused_reason),
    updated_at = sqlc.arg(updated_at)
WHERE drain_id = sqlc.arg(drain_id)
  AND (paused_reason IS NULL OR paused_reason = '');
