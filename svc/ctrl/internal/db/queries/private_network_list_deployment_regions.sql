-- name: ListPrivateNetworkDeploymentRegions :many
-- ListPrivateNetworkDeploymentRegions returns the regions on the platform where
-- each given deployment is meant to run.
SELECT dt.deployment_id, r.name AS region
FROM deployment_topology dt
INNER JOIN regions r ON r.id = dt.region_id
WHERE dt.deployment_id IN (sqlc.slice(deployment_ids))
    AND dt.desired_status = 'running'
    AND r.platform = sqlc.arg(platform)
ORDER BY dt.deployment_id, r.name;
