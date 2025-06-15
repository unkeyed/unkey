-- name: FindPermissionBySlugAndWorkspaceID :one
SELECT *
FROM permissions
WHERE slug = sqlc.arg(slug)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;