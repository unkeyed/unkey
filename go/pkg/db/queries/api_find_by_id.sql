-- name: FindApiByID :one
SELECT * FROM apis WHERE id = ? AND deleted_at_m IS NULL;
