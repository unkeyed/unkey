-- name: FindKeyByID :one
SELECT
    sqlc.embed(k),
    sqlc.embed(i)
FROM `keys` k
LEFT JOIN identities i ON k.identity_id = i.id
WHERE k.id = sqlc.arg(id);
