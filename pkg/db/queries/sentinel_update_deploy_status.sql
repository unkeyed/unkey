-- name: UpdateSentinelDeployStatus :exec
-- UpdateSentinelDeployStatus updates only the deploy status field.
-- Used after convergence check or rollback completes.
UPDATE sentinels SET
  deploy_status = sqlc.arg(deploy_status),
  updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id);
