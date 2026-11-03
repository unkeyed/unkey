-- name: StopDeploymentsByEnvironment :exec
-- StopDeploymentsByEnvironment also notifies each region in the same write.
UPDATE deployments d
LEFT JOIN deployment_topology dt ON dt.deployment_id = d.id
SET d.desired_state = 'stopped', d.updated_at = sqlc.arg(updated_at),
    dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, sqlc.arg(updated_at))
WHERE d.environment_id = sqlc.arg(environment_id);
