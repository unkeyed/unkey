-- name: DeleteDeploymentInstances :exec
DELETE FROM instances
WHERE deployment_id = ?  and cluster_id = ?;
