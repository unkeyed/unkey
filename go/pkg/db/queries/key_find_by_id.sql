-- name: FindKeyByID :one
SELECT * FROM `keys` k
WHERE k.id = sqlc.arg(id);
