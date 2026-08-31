-- name: FindPermissionByIdOrSlug :one
-- FindPermissionByIdOrSlug resolves a permission within a workspace so the
-- caller can authorize access against the permission's actual project.
SELECT *
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg(search) OR slug = sqlc.arg(search));
