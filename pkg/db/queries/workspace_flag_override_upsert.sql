-- name: UpsertWorkspaceFlagOverride :exec
-- UpsertWorkspaceFlagOverride makes retries safe and keeps one override per
-- workspace and flag. The caller validates enrollment permission first.
INSERT INTO flag_overrides (workspace_id, flag_slug, enabled)
VALUES (sqlc.arg(workspace_id), sqlc.arg(flag_slug), sqlc.arg(enabled))
ON DUPLICATE KEY UPDATE enabled = VALUES(enabled);
