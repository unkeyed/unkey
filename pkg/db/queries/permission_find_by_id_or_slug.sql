-- name: FindPermissionByIdOrSlug :one
-- FindPermissionByIdOrSlug resolves IDs across a workspace while slugs resolve
-- only in the selected project because slug uniqueness is project-scoped.
-- An ID match wins when search also matches a slug because ORDER BY ranks IDs
-- first. For example, search "perm_admin" returns the row with that ID instead
-- of a selected-project permission whose slug is "perm_admin".
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m
FROM permissions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (
    id = sqlc.arg(search)
    OR (project_id = sqlc.arg(project_id) AND slug = sqlc.arg(search))
  )
ORDER BY id = sqlc.arg(search) DESC
LIMIT 1;
