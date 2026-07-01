-- name: UpsertAppRegionalSettings :exec
INSERT INTO app_regional_settings (
    workspace_id,
    app_id,
    environment_id,
    region_id,
    replicas,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(workspace_id),
    sqlc.arg(app_id),
    sqlc.arg(environment_id),
    sqlc.arg(region_id),
    sqlc.arg(replicas),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
)
ON DUPLICATE KEY UPDATE
    replicas = VALUES(replicas),
    updated_at = VALUES(updated_at);
