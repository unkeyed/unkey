-- name: CountKeysByKeyAuth :one
SELECT
    count(id)
FROM
    `keys`
WHERE
    key_auth_id = sqlc.arg("key_auth_id");
