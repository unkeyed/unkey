
-- name: FindKeyByHash :one
SELECT
    *
FROM `keys`
WHERE hash = sqlc.arg(hash);
