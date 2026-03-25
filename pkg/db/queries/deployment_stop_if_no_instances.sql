-- name: StopDeploymentIfNoInstances :exec
UPDATE deployments d
LEFT JOIN instances i ON i.deployment_id = d.id
SET d.status = 'stopped', d.updated_at = sqlc.arg(updated_at)
WHERE d.id = sqlc.arg(id)
  AND d.desired_state IN ('standby', 'archived')
  AND i.deployment_id IS NULL;
