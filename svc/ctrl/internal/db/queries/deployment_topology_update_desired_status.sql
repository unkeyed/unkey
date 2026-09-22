-- name: UpdateDeploymentTopologyDesiredStatus :exec
-- UpdateDeploymentTopologyDesiredStatus updates desired status and advances its revision atomically.
UPDATE `deployment_topology`
SET desired_status = sqlc.arg(desired_status), revision = revision + 1, updated_at = sqlc.arg(updated_at)
WHERE deployment_id = sqlc.arg(deployment_id) AND region_id = sqlc.arg(region_id);
