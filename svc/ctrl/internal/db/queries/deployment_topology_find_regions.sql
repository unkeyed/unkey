-- name: FindDeploymentRegions :many
-- Returns all regions where a deployment is configured.
-- Used for fan-out: when a deployment changes, emit state_change to each region.
SELECT r.*
FROM `deployment_topology` dt
INNER JOIN `regions` r ON r.id = dt.region_id
WHERE dt.deployment_id = sqlc.arg(deployment_id);
