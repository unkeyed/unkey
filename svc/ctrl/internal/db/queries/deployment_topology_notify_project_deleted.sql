-- name: NotifyDeploymentTopologiesByProject :exec
UPDATE deployment_topology dt
JOIN deployments d ON d.id = dt.deployment_id
SET dt.desired_status = 'stopped',
    dt.updated_at = GREATEST(COALESCE(dt.updated_at, 0) + 1, sqlc.arg(updated_at))
WHERE d.project_id = sqlc.arg(project_id);
