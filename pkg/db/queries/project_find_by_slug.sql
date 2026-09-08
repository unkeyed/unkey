-- name: FindProjectBySlug :one
SELECT projects.pk, projects.id, projects.workspace_id, projects.name, projects.slug, projects.depot_project_id, projects.delete_protection, projects.created_at, projects.updated_at
FROM projects
WHERE slug = ?
LIMIT 1;
