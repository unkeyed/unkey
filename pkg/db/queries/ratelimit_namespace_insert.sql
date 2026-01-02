-- name: InsertRatelimitNamespace :exec
INSERT INTO
    `ratelimit_namespaces` (
        id,
        workspace_id,
        name,
        created_at_m,
        updated_at_m,
        deleted_at_m
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
