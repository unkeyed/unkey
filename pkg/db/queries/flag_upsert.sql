-- name: UpsertFlag :exec
-- UpsertFlag creates or replaces a definition. Only tests use it, to change the
-- default of a fixed slug between cases.
INSERT INTO flags (slug, description, default_enabled, allow_opt_in, allow_opt_out)
VALUES (sqlc.arg(slug), sqlc.arg(description), sqlc.arg(default_enabled),
    sqlc.arg(allow_opt_in), sqlc.arg(allow_opt_out))
ON DUPLICATE KEY UPDATE description = VALUES(description),
    default_enabled = VALUES(default_enabled),
    allow_opt_in = VALUES(allow_opt_in), allow_opt_out = VALUES(allow_opt_out);
