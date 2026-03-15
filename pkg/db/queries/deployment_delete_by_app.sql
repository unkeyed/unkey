-- name: DeleteDeploymentsByAppId :exec
DELETE FROM deployments WHERE app_id = sqlc.arg(app_id);
