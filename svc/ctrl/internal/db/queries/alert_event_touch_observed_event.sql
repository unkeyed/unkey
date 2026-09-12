-- name: TouchAlertEventFromObservedEvent :exec
UPDATE alert_events
SET last_seen_at = GREATEST(last_seen_at, sqlc.arg(observed_at)),
    last_observed_event_at = GREATEST(COALESCE(last_observed_event_at, 0), sqlc.arg(observed_at)),
    observed_value = sqlc.arg(observed_value),
    updated_at = sqlc.arg(observed_at)
WHERE id = sqlc.arg(id) AND status = 'open';
