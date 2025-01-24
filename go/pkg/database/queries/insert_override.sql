-- name: InsertOverride :exec
INSERT INTO
    `ratelimit_overrides` (
        id,
        workspace_id,
        namespace_id,
        identifier,
        `limit`,
        duration,
        async,
        created_at
    )
VALUES
    (
        sqlc.arg("id"),
        sqlc.arg("workspace_id"),
        sqlc.arg("namespace_id"),
        sqlc.arg("identifier"),
        sqlc.arg("limit"),
        sqlc.arg("duration"),
        false,
        now()
    )
