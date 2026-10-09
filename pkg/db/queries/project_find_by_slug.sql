-- name: FindProjectBySlug :one
SELECT projects.pk, projects.id, projects.workspace_id, projects.name, projects.slug, projects.depot_project_id, projects.delete_protection, projects.created_at, projects.updated_at, projects.deleted_at_m
FROM projects
WHERE slug = ?
  AND deleted_at_m IS NULL
LIMIT 1;
