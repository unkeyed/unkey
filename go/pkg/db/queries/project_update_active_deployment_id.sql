-- name: UpdateProjectActiveDeploymentId :exec
UPDATE projects
SET active_deployment_id = ?, updated_at = ?
WHERE id = ?;
