-- name: FindLatestReadyDeploymentByAppAndEnv :one
SELECT id
FROM deployments
WHERE app_id = sqlc.arg(app_id)
  AND environment_id = sqlc.arg(environment_id)
  AND status = 'ready'
  AND id != sqlc.arg(exclude_id)
ORDER BY created_at DESC
LIMIT 1;
