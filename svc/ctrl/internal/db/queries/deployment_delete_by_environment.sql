-- name: DeleteDeploymentsByEnvironmentId :exec
DELETE FROM deployments WHERE environment_id = sqlc.arg(environment_id);
