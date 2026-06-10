-- name: ListAppsByProject :many
SELECT sqlc.embed(apps)
FROM apps
WHERE project_id = sqlc.arg(project_id)
  AND deletion_id IS NULL
ORDER BY created_at ASC;
