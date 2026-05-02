-- name: GetLogDrainState :one
-- Reads the per-drain operational state row written by
-- RecordLogDrainFailure / RecordLogDrainSuccess. Used after a failure has
-- landed to decide whether the consecutive_failures counter has crossed
-- the auto-pause threshold; PauseLogDrain then sets paused_reason.
SELECT drain_id, last_delivery_at, last_attempt_at, last_error,
       consecutive_failures, paused_reason, total_records_delivered, updated_at
FROM log_drain_state
WHERE drain_id = ?;
