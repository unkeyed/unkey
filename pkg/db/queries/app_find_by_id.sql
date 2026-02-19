-- name: FindAppById :one
SELECT sqlc.embed(apps)
FROM apps
WHERE id = sqlc.arg(id);
