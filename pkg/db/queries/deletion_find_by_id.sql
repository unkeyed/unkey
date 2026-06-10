-- name: FindDeletionById :one
SELECT *
FROM `deletions`
WHERE id = ?;
