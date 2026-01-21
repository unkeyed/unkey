-- name: FindDeploymentRegions :many
-- Returns all regions where a deployment is configured.
-- Used for fan-out: when a deployment changes, emit state_change to each region.
SELECT region
FROM `deployment_topology`
WHERE deployment_id = sqlc.arg(deployment_id);
