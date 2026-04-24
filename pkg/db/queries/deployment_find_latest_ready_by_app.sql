-- name: FindLatestReadyDeploymentByApp :one
SELECT * FROM `deployments`
WHERE app_id = sqlc.arg(app_id)
  AND status = 'ready'
ORDER BY created_at DESC
LIMIT 1;
