-- name: ClearAppCurrentDeploymentByEnvironment :exec
UPDATE apps a
JOIN deployments d ON d.id = a.current_deployment_id
SET a.current_deployment_id = NULL, a.updated_at = sqlc.arg(updated_at)
WHERE d.environment_id = sqlc.arg(environment_id);
