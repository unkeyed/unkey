-- name: GetKeyByHash :one
SELECT * FROM `keys`
WHERE hash = sqlc.arg(hash);
