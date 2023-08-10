-- name: CreateKeyAuth :exec
INSERT INTO
    `key_auth` (id, workspace_id)
VALUES
    (
        sqlc.arg("id"),
        sqlc.arg("workspace_id")
    )