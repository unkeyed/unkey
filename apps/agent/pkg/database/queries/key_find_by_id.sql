-- name: FindKeyById :one
SELECT
    *
FROM
    `keys`
WHERE
    id = sqlc.arg("id");
