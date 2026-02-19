-- name: UpdateAppDeployments :exec
UPDATE apps
SET
  live_deployment_id = sqlc.arg(live_deployment_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
