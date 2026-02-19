-- name: ListSiblingAppsInEnvironment :many
SELECT sqlc.embed(apps)
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND id != sqlc.arg(exclude_app_id)
ORDER BY created_at ASC;
