-- name: ResolveOpenAlertEventsByApp :exec
-- ResolveOpenAlertEventsByApp closes detector alerts for every environment
-- before app deletion fans out its environment cleanup.
UPDATE alert_events
SET status = 'resolved',
    resolved_at = sqlc.arg(resolved_at),
    resolution_message = 'Deployment stopped',
    updated_at = sqlc.arg(updated_at)
WHERE app_id = sqlc.arg(app_id)
  AND status = 'open';
