-- name: UpsertFlag :exec
-- UpsertFlag creates or replaces a definition. Only tests use it, to change the
-- default of a fixed slug between cases.
INSERT INTO flags (slug, description, default_value, allow_enable, allow_disable)
VALUES (sqlc.arg(slug), sqlc.arg(description), sqlc.arg(default_value),
    sqlc.arg(allow_enable), sqlc.arg(allow_disable))
ON DUPLICATE KEY UPDATE description = VALUES(description),
    default_value = VALUES(default_value),
    allow_enable = VALUES(allow_enable), allow_disable = VALUES(allow_disable);
