-- name: DeleteDeploymentInstances :exec
DELETE FROM instances
WHERE deployment_id = ? AND region = ?;
