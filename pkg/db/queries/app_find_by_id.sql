-- name: FindAppById :one
SELECT *
FROM apps
WHERE id = sqlc.arg(id);
