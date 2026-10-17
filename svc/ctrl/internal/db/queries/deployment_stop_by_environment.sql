-- name: StopDeploymentsByEnvironment :exec
UPDATE deployments SET desired_state = 'stopped', updated_at = sqlc.arg(updated_at)
WHERE environment_id = sqlc.arg(environment_id);
