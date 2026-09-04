-- name: FindPermissionBySlugAndProjectID :one
-- FindPermissionBySlugAndProjectID resolves a slug in one project while the
-- workspace predicate preserves tenant isolation.
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE slug = sqlc.arg(slug)
AND project_id = sqlc.arg(project_id)
AND workspace_id = sqlc.arg(workspace_id)
LIMIT 1;
