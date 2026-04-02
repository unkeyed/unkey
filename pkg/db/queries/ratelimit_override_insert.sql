-- name: InsertRatelimitOverride :exec
INSERT INTO ratelimit_overrides (
    id,
    workspace_id,
    namespace_id,
    identifier,
    `limit`,
    duration,
    created_at_m
)
VALUES (
    sqlc.arg("id"),
    sqlc.arg("workspace_id"),
    sqlc.arg("namespace_id"),
    sqlc.arg("identifier"),
    sqlc.arg("limit"),
    sqlc.arg("duration"),
    sqlc.arg("created_at")
)
ON DUPLICATE KEY UPDATE
    `limit` = VALUES(`limit`),
    duration = VALUES(duration),
    updated_at_m = sqlc.arg('updated_at')
