-- name: FindProjectBySlug :one
SELECT *
FROM projects
WHERE slug = ?
  AND deletion_id IS NULL
LIMIT 1;
