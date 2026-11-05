-- name: FindDeviceSessionByID :one
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE id = sqlc.arg(id);

-- name: FindDeviceSessionByRefreshTokenHash :one
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE refresh_token_hash = sqlc.arg(refresh_token_hash);

-- name: FindDeviceSessionByPreviousRefreshTokenHash :one
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE previous_refresh_token_hash = sqlc.arg(previous_refresh_token_hash);

-- name: ListDeviceSessionsByWorkspace :many
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;

-- name: ListDeviceSessionsByUser :many
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;

-- name: FindUnrevokedDeviceSessionByRefreshFamily :one
SELECT
    pk,
    id,
    kind,
    workspace_id,
    user_id,
    user_name,
    permissions,
    device_name,
    oauth_client_id,
    resource,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM device_sessions
WHERE refresh_family = sqlc.arg(refresh_family)
  AND revoked_at IS NULL
LIMIT 1;
