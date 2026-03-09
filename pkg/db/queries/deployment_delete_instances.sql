-- name: DeleteDeploymentInstances :exec
DELETE FROM instances
WHERE deployment_id = sqlc.arg(deployment_id) AND region_id = sqlc.arg(region_id);
