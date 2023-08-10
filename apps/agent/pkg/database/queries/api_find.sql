-- name: FindApi :one
SELECT
    *
FROM
    apis
WHERE
    id = sqlc.arg("id");
