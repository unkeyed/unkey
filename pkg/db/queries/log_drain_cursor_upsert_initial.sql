-- name: UpsertLogDrainCursorInitial :exec
-- Establishes a cursor for a brand-new group. Uses INSERT IGNORE so two
-- replicas racing to bootstrap the same group never overwrite each other;
-- the loser silently no-ops and reads the existing cursor on the next tick.
INSERT IGNORE INTO log_drain_cursors (group_key, inserted_at_ms, fingerprint, updated_at)
VALUES (?, ?, ?, ?);
