-- name: StopDeploymentTopologiesByEnvironment :exec
UPDATE deployment_topology dt
JOIN deployments d ON d.id = dt.deployment_id
SET dt.desired_status = 'stopped',
    dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, CAST(sqlc.narg(updated_at) AS SIGNED))
WHERE d.environment_id = sqlc.arg(environment_id);
