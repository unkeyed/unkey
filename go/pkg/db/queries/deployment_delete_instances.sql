-- name: DeleteDeploymentInstances :exec
DELETE FROM instances
WHERE deployment_id = ?  and region = ? and shard = ?;
