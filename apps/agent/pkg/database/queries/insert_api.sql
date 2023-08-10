-- name: InsertApi :exec
INSERT INTO
    `apis` (
        id,
        workspace_id,
        name,
        ip_whitelist,
        auth_type,
        key_auth_id
    )
VALUES
    (
        sqlc.arg("id"),
        sqlc.arg("workspace_id"),
        sqlc.arg("name"),
        sqlc.arg("ip_whitelist"),
        sqlc.arg("auth_type"),
        sqlc.arg("key_auth_id")
    )