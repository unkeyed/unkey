-- name: FindDeploymentRegions :many
-- Returns all regions where a deployment is configured.
-- Used for fan-out: when a deployment changes, emit state_change to each region.
SELECT r.name
FROM `deployment_topology` dt
INNER JOIN `regions` r ON dt.region_id = r.id
WHERE dt.deployment_id = sqlc.arg(deployment_id);
