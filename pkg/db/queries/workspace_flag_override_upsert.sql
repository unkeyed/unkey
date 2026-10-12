-- name: UpsertWorkspaceFlagOverride :exec
-- UpsertWorkspaceFlagOverride makes retries safe and keeps one override per
-- workspace and flag. The caller validates enrollment permission first.
INSERT INTO flag_overrides (workspace_id, flag_slug, value)
VALUES (sqlc.arg(workspace_id), sqlc.arg(flag_slug), sqlc.arg(value))
ON DUPLICATE KEY UPDATE value = VALUES(value);
