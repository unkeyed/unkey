-- name: FindManyPermissionsByIdOrSlug :many
SELECT *
FROM permissions
WHERE workspace_id = ? AND (id IN (sqlc.slice('ids')) OR slug IN (sqlc.slice('ids')));
