-- name: TouchAlertEventLastSeen :exec
-- TouchAlertEventLastSeen records the latest complete anomalous window without
-- changing the baseline snapshot captured when the alert opened.
UPDATE alert_events
SET last_seen_at = sqlc.arg(last_seen_at),
    observed_value = sqlc.arg(observed_value),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND status = 'open';
