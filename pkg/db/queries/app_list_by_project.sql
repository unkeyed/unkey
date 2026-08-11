-- name: ListAppsByProject :many
SELECT
  apps.id,
  apps.name,
  apps.slug,
  apps.source_type,
  apps.current_deployment_id,
  apps.is_rolled_back,
  apps.delete_protection,
  apps.created_at,
  apps.updated_at,
  grc.repository_full_name AS repository_full_name,
  grc.default_branch AS github_default_branch,
  ads.image_reference AS docker_image_reference
FROM apps
LEFT JOIN github_repo_connections grc ON grc.app_id = apps.id
LEFT JOIN app_docker_sources ads ON ads.app_id = apps.id
WHERE apps.project_id = sqlc.arg(project_id)
  AND apps.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(apps.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.name) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.slug) LIKE LOWER(sqlc.narg(search)))
ORDER BY apps.id ASC
LIMIT ?;
