-- name: UpsertAppRuntimeSettingsSentinelConfig :exec
-- Writes only sentinel_config, creating the row when missing. All other
-- columns keep their current values.
INSERT INTO app_runtime_settings (
    workspace_id,
    app_id,
    environment_id,
    sentinel_config,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(sentinel_config),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    sentinel_config = VALUES(sentinel_config),
    updated_at = VALUES(updated_at);
