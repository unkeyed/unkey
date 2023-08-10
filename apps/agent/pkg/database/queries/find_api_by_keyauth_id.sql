-- name: FindApiByKeyAuthId :one
SELECT
    *
FROM
    `apis`
WHERE
    key_auth_id = sqlc.arg("keyAuthId");
