-- name: FindEnvironmentById :one
SELECT *
FROM environments
WHERE id = sqlc.arg(id);
