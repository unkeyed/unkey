-- name: UpdateProjectLiveDeploymentId :exec
UPDATE projects
SET live_deployment_id = ?, updated_at = ?
WHERE id = ?;
