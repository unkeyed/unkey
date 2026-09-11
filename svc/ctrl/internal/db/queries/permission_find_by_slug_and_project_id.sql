-- name: FindPermissionBySlugAndProjectID :one
-- FindPermissionBySlugAndProjectID resolves a duplicate insert to the existing
-- row in the requested project. The workspace filter preserves tenant isolation.
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE slug = sqlc.arg(slug)
  AND workspace_id = sqlc.arg(workspace_id)
  AND project_id = sqlc.arg(project_id)
LIMIT 1;
