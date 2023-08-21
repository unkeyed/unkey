-- name: FindWorkspace :one
SELECT
    *
FROM
    `workspaces`
WHERE
    id = sqlc.arg("id");
