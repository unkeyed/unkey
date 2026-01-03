-- name: UpdateProjectDeployments :exec
UPDATE projects
SET
  live_deployment_id = sqlc.arg(live_deployment_id),
  is_rolled_back = sqlc.arg(is_rolled_back),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
