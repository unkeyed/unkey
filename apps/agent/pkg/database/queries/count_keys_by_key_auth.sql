-- name: CountKeysByKeyAuth :one
SELECT
    count(*)
FROM
    `keys`
WHERE
    key_auth_id = sqlc.arg("key_auth_id");
