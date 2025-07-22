-- name: FindPermissionByIdOrSlug :one
SELECT *
FROM permissions
WHERE workspace_id = ? AND (id = ? OR slug = ?);
