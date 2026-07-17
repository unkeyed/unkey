-- name: ListDeploymentRegions :many
-- Region names a deployment is configured to run in. Reads the covering
-- unique(deployment_id, region_id) key on deployment_topology, joined to
-- regions for the human-readable name.
SELECT DISTINCT r.name AS region
FROM deployment_topology dt
JOIN regions r ON r.id = dt.region_id
WHERE dt.deployment_id = sqlc.arg(deployment_id)
ORDER BY r.name;

-- name: ListDeploymentRegionsByIds :many
-- Batch form keyed by deployment id: fetches configured region names for a
-- whole page of deployments in one query, so listDeployments avoids an N+1.
SELECT DISTINCT dt.deployment_id AS deployment_id, r.name AS region
FROM deployment_topology dt
JOIN regions r ON r.id = dt.region_id
WHERE dt.deployment_id IN (sqlc.slice('deployment_ids'))
ORDER BY dt.deployment_id, r.name;
