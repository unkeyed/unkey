-- name: UpdateAppDeployments :exec
UPDATE apps
SET
  current_deployment_id = sqlc.arg(current_deployment_id),
  is_rolled_back = sqlc.arg(is_rolled_back),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(app_id);
