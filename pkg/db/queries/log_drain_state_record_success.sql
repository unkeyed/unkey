-- name: RecordLogDrainSuccess :exec
INSERT INTO log_drain_state (
    drain_id,
    last_delivery_at,
    last_attempt_at,
    last_error,
    consecutive_failures,
    paused_reason,
    total_records_delivered,
    updated_at
) VALUES (?, ?, ?, NULL, 0, NULL, ?, ?)
ON DUPLICATE KEY UPDATE
    last_delivery_at = VALUES(last_delivery_at),
    last_attempt_at = VALUES(last_attempt_at),
    last_error = NULL,
    consecutive_failures = 0,
    paused_reason = NULL,
    total_records_delivered = log_drain_state.total_records_delivered + VALUES(total_records_delivered),
    updated_at = VALUES(updated_at);
