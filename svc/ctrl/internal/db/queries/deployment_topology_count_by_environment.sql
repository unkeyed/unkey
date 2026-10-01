-- name: CountDeploymentTopologiesByEnvironment :one
SELECT COUNT(*) FROM deployment_topology dt
JOIN deployments d ON d.id = dt.deployment_id
WHERE d.environment_id = sqlc.arg(environment_id);
