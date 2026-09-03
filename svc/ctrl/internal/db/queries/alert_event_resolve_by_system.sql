-- name: ResolveAlertEventBySystem :execrows
-- ResolveAlertEventBySystem closes an alert only while it is open. The status
-- guard makes retries and concurrent reconciliation safe no-ops.
UPDATE alert_events
SET status = 'resolved',
    resolved_at = sqlc.arg(resolved_at),
    resolution_message = sqlc.arg(resolution_message),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND status = 'open';
