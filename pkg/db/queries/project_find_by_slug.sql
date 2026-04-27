-- name: FindProjectBySlug :one
SELECT *
FROM projects
WHERE slug = ?
LIMIT 1;
