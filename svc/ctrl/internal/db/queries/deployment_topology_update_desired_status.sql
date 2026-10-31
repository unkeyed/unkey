-- name: UpdateDeploymentTopologyDesiredStatus :exec
-- UpdateDeploymentTopologyDesiredStatus also updates deployment intent while
-- legacy workers still stop compute through regional topology writes.
UPDATE deployment_topology dt
JOIN deployments d ON d.id = dt.deployment_id
SET d.desired_state = CASE WHEN sqlc.arg(desired_status) = 'stopped' THEN 'stopped' ELSE 'running' END,
    d.status = CASE
        WHEN sqlc.arg(desired_status) = 'running' AND d.status IN ('failed', 'cancelled', 'superseded', 'skipped') THEN 'deploying'
        ELSE d.status
    END,
    dt.desired_status = sqlc.arg(desired_status),
    dt.updated_at = sqlc.arg(updated_at)
WHERE dt.deployment_id = sqlc.arg(deployment_id) AND dt.region_id = sqlc.arg(region_id);
