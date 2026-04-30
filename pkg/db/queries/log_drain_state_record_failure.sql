-- name: RecordLogDrainFailure :exec
-- Increments consecutive_failures and stores the verbatim provider error.
-- The application sets paused_reason once the threshold is crossed;
-- keeping pause logic in code (not SQL) means the threshold can change
-- without a schema migration.
INSERT INTO log_drain_state (
    drain_id,
    last_delivery_at,
    last_attempt_at,
    last_error,
    consecutive_failures,
    paused_reason,
    total_records_delivered,
    updated_at
) VALUES (?, NULL, ?, ?, 1, ?, 0, ?)
ON DUPLICATE KEY UPDATE
    last_attempt_at = VALUES(last_attempt_at),
    last_error = VALUES(last_error),
    consecutive_failures = log_drain_state.consecutive_failures + 1,
    paused_reason = VALUES(paused_reason),
    updated_at = VALUES(updated_at);
