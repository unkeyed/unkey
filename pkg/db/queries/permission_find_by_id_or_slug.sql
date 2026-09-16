-- name: FindPermissionByIdOrSlug :one
-- FindPermissionByIdOrSlug resolves a permission within a workspace so the
-- caller can authorize access against the permission's actual project.
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg(search) OR slug = sqlc.arg(search));
