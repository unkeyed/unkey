-- name: ReassignDomain :exec
UPDATE domains
SET
  workspace_id = sqlc.arg(target_workspace_id),
  deployment_id = sqlc.arg(target_deployment_id),
  is_rolled_back = sqlc.arg(is_rolled_back),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
