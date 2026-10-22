-- name: ListInstanceContainerStatusesByDeploymentIds :many
SELECT instances.deployment_id, instances.container_status FROM instances
WHERE workspace_id = sqlc.arg(workspace_id)
  AND deployment_id IN (sqlc.slice('deployment_ids'));
