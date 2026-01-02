-- name: FindPermissionByNameAndWorkspaceID :one
SELECT *
FROM permissions
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
