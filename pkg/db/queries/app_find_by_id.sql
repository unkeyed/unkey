-- name: FindAppById :one
SELECT *
FROM apps
WHERE id = sqlc.arg(id)
  AND deletion_id IS NULL;
