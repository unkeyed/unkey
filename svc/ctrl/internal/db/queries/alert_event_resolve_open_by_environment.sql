-- name: ResolveOpenAlertEventsByEnvironment :exec
-- ResolveOpenAlertEventsByEnvironment closes detector alerts before their
-- environment metadata is deleted, preventing invisible open inbox counts.
UPDATE alert_events
SET status = 'resolved',
    resolved_at = sqlc.arg(resolved_at),
    resolution_message = 'Deployment stopped',
    updated_at = sqlc.arg(updated_at)
WHERE environment_id = sqlc.arg(environment_id)
  AND status = 'open';
