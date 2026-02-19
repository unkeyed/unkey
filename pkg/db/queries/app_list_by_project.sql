-- name: ListAppsByProject :many
SELECT sqlc.embed(apps)
FROM apps
WHERE project_id = sqlc.arg(project_id)
ORDER BY created_at ASC;
