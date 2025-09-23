-- name: ReassignDomain :exec
UPDATE domains
SET
  workspace_id = sqlc.arg(target_workspace_id),
  deployment_id = sqlc.arg(deployment_id),
  rolled_back_deployment_id = sqlc.arg(rolled_back_deployment_id),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
