-- name: CountDeploymentsByAppId :one
-- CountDeploymentsByAppId counts an app's deployments. Only tests use this query.
SELECT COUNT(*)
FROM `deployments`
WHERE app_id = sqlc.arg(app_id);
