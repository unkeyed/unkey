-- name: ListKeysByKeyAuthId :many
SELECT
    *
FROM
    `keys`
WHERE
    key_auth_id = sqlc.arg('key_auth_id')
ORDER BY
    created_at ASC
LIMIT
    ? OFFSET ?;