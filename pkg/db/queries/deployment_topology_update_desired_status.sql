-- name: UpdateDeploymentTopologyDesiredStatus :exec
-- UpdateDeploymentTopologyDesiredStatus updates the desired_status of a topology entry.
UPDATE `deployment_topology`
SET desired_status = sqlc.arg(desired_status), updated_at = sqlc.arg(updated_at)
WHERE deployment_id = sqlc.arg(deployment_id) AND region_id = sqlc.arg(region_id);
