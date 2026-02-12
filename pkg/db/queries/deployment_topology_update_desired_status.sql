-- name: UpdateDeploymentTopologyDesiredStatus :exec
-- UpdateDeploymentTopologyDesiredStatus updates the desired_status and version of a topology entry.
-- A new version is required so that WatchDeployments picks up the change.
UPDATE `deployment_topology`
SET desired_status = sqlc.arg(desired_status), version = sqlc.arg(version), updated_at = sqlc.arg(updated_at)
WHERE deployment_id = sqlc.arg(deployment_id) AND region = sqlc.arg(region);
