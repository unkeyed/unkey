-- name: ListDeploymentsByEnvironmentIdAndStatus :many
SELECT * FROM `deployments`
WHERE environment_id = sqlc.arg(environment_id)
  AND status = sqlc.arg(status);
