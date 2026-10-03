-- name: UpsertWorkspaceFlagOverride :exec
-- UpsertWorkspaceFlagOverride makes retries safe and keeps one override per
-- workspace and flag. The caller validates type and enrollment permission first.
INSERT INTO workspace_flag_overrides (workspace_id, flag_id, value)
VALUES (sqlc.arg(workspace_id), sqlc.arg(flag_id), sqlc.arg(value))
ON DUPLICATE KEY UPDATE value = VALUES(value);
