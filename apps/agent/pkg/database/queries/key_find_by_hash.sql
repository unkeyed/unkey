-- name: FindKeyByHash :one
SELECT
    *
FROM
    `keys`
WHERE
    hash = sqlc.arg("hash")
AND deleted_at IS NULL