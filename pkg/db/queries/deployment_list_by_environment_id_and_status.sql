-- name: ListDeploymentsByEnvironmentIdAndStatus :many
SELECT * FROM `deployments`
WHERE environment_id = sqlc.arg(environment_id)
  AND status = sqlc.arg(status)
  AND created_at < sqlc.arg(created_before)
  AND (updated_at IS null OR updated_at < sqlc.arg(updated_before) )
