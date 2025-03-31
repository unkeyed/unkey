-- name: InsertRatelimitOverride :exec
INSERT INTO
    `ratelimit_overrides` (
        id,
        workspace_id,
        namespace_id,
        identifier,
        `limit`,
        duration,
        async,
        created_at_m
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
         sqlc.arg("created_at")
    )
