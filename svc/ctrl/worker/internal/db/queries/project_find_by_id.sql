-- name: FindProjectById :one
SELECT *
FROM projects
WHERE id = ?;
