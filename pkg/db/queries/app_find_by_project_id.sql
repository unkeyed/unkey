-- name: FindAppsByProjectId :many
SELECT id, slug
FROM apps
WHERE project_id = ?;
