-- name: LockAppConnection :one
SELECT c.id, c.project_id, c.app_id, c.environment_id, c.resource_id, c.name
FROM app_connections c
INNER JOIN connection_app_targets t ON t.connection_id = c.id
WHERE c.id = sqlc.arg(id)
    AND c.workspace_id = sqlc.arg(workspace_id)
    AND c.resource_type = 'app'
FOR UPDATE;
