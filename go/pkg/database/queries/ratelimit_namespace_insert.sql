-- name: InsertRatelimitNamespace :exec
INSERT INTO
    `ratelimit_namespaces` (
        id,
        workspace_id,
        name,
        created_at,
        updated_at,
        deleted_at
        )
VALUES
    (
        sqlc.arg("id"),
        sqlc.arg("workspace_id"),
        sqlc.arg("name"),
         sqlc.arg(created_at),
        NULL,
        NULL
    )
;
