-- name: FailDeploymentAndStop :exec
-- FailDeploymentAndStop records failure and stop intent before regional cleanup
-- can report completion. It preserves cancellation outcomes and ready deployments.
UPDATE deployments d
LEFT JOIN deployment_topology dt ON dt.deployment_id = d.id
SET d.status = CASE WHEN d.status IN (sqlc.slice('progressing_statuses')) THEN 'failed' ELSE d.status END,
    d.desired_state = 'stopped',
    d.updated_at = sqlc.narg(updated_at),
    dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, COALESCE(sqlc.narg(updated_at), 0))
WHERE d.id = sqlc.arg(id)
  AND d.status NOT IN ('ready', 'stopped');
