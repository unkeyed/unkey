-- name: ListAppsByProject :many
SELECT
  apps.*,
  grc.repository_full_name AS repository_full_name,
  grc.default_branch AS github_default_branch
FROM apps
LEFT JOIN github_repo_connections grc ON grc.app_id = apps.id
WHERE apps.project_id = sqlc.arg(project_id)
  AND apps.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(apps.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.name) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.slug) LIKE LOWER(sqlc.narg(search)))
ORDER BY apps.id ASC
LIMIT ?;
