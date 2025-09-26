-- name: FindPermissionByIdOrSlug :one
SELECT *
FROM permissions
WHERE workspace_id = ? AND (id = sqlc.arg('search') OR slug = sqlc.arg('search'));
