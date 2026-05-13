-- name: UpdateDeploymentStatusIfActive :exec
-- Transition a deployment's status only when its current status is still
-- "active" (non-terminal). Prevents the Deploy handler's compensation
-- stack from overwriting a status that was set intentionally by the dedup
-- path (e.g. superseded) or by a successful completion (ready).
UPDATE deployments
SET status = sqlc.arg('status'), updated_at = sqlc.arg('updated_at')
WHERE id = sqlc.arg('id')
  AND status NOT IN ('ready', 'failed', 'superseded', 'skipped', 'stopped', 'cancelled');
