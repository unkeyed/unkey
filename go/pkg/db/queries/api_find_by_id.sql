-- name: FindApiByID :one
SELECT * FROM apis WHERE id = ?;
