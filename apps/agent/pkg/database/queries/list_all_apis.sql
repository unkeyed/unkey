-- name: ListAllApis :many
SELECT
    *
FROM
    `apis`
ORDER BY
    id ASC
LIMIT
    ? OFFSET ?;