-- name: UpdateProjectDeployments :exec
UPDATE projects
SET
  live_deployment_id = sqlc.arg(live_deployment_id),
  rolled_back_deployment_id = sqlc.arg(rolled_back_deployment_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
