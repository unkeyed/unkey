-- name: FindInstanceById :one
SELECT * FROM instance WHERE id = ?;
