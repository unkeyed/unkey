-- name: ListDeploymentRegions :many
SELECT DISTINCT r.name AS region
FROM deployment_topology dt
JOIN regions r ON r.id = dt.region_id
WHERE dt.workspace_id = sqlc.arg(workspace_id)
  AND dt.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.name;

-- name: ListDeploymentRegionsByIds :many
SELECT DISTINCT dt.deployment_id AS deployment_id, r.name AS region
FROM deployment_topology dt
JOIN regions r ON r.id = dt.region_id
WHERE dt.workspace_id = sqlc.arg(workspace_id)
  AND dt.deployment_id IN (sqlc.slice('deployment_ids'))
ORDER BY dt.deployment_id, r.name;
