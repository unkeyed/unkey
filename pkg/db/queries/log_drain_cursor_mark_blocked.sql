-- name: MarkLogDrainCursorBlocked :exec
-- Sets blocked=true on a drain's cursor row. The coordinator's
-- in-memory groupMinCursor (and any future MIN query) excludes blocked
-- drains so the group's read watermark advances past a persistently
-- failing drain instead of stalling there indefinitely. The drain
-- itself stays in the log_drains table; resume is a dashboard-driven
-- UPDATE that flips blocked back to false and clears blocked_reason.
UPDATE log_drain_cursors
SET blocked = true,
    blocked_reason = sqlc.arg(blocked_reason),
    updated_at = sqlc.arg(updated_at)
WHERE drain_id = sqlc.arg(drain_id)
  AND blocked = false;
