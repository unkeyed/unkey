-- name: FindPermissionByNameAndWorkspaceID :one
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE name = sqlc.arg(name)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
