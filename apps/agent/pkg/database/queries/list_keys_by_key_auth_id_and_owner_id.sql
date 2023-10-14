-- name: ListKeysByKeyAuthIdAndOwnerId :many
SELECT
    *
FROM
    `keys`
WHERE
    key_auth_id = sqlc.arg('key_auth_id')
    AND owner_id = sqlc.arg('owner_id')
    AND deleted_at IS NULL
ORDER BY
    created_at ASC
LIMIT
    ? OFFSET ?;