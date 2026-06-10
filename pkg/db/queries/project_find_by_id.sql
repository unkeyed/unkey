-- name: FindProjectById :one
SELECT *
FROM projects
WHERE id = ?
  AND deletion_id IS NULL;
