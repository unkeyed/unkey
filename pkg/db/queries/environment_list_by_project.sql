-- name: ListEnvironmentsByProject :many
SELECT sqlc.embed(environments)
FROM environments
WHERE project_id = sqlc.arg(project_id)
ORDER BY created_at ASC;
