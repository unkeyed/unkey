-- name: UpsertLogDrainCursorInitial :exec
-- Bootstraps a per-drain cursor for a brand-new drain. INSERT IGNORE so
-- two replicas racing to initialise the same drain do not stomp each
-- other; the loser silently no-ops and reads the existing cursor on the
-- next tick. The bootstrap watermark is set by the caller to (now -
-- BatchWindow, '') so a freshly-created drain doesn't replay the
-- 90-day ClickHouse retention window on first delivery.
INSERT IGNORE INTO log_drain_cursors (
    drain_id, group_key, time_ms, last_id, blocked, blocked_reason, updated_at
) VALUES (?, ?, ?, ?, false, NULL, ?);
