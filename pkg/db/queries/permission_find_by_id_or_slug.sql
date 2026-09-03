-- name: FindPermissionByIdOrSlug :one
-- FindPermissionByIdOrSlug resolves IDs from any project in the workspace and
-- resolves slugs only from the requested project.
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    id = sqlc.arg(search)
    OR (
      project_id = sqlc.arg(project_id)
      AND slug = sqlc.arg(search)
    )
  );
