-- name: FindUserDeviceSessionByID :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE id = sqlc.arg(id);

-- name: FindUserDeviceSessionByRefreshTokenHash :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE refresh_token_hash = sqlc.arg(refresh_token_hash);

-- name: FindUserDeviceSessionByPreviousRefreshTokenHash :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE previous_refresh_token_hash = sqlc.arg(previous_refresh_token_hash);

-- name: ListUserDeviceSessionsByWorkspace :many
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;

-- name: ListUserDeviceSessionsByUser :many
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;

-- name: FindUnrevokedUserDeviceSessionByRefreshFamily :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    kind,
    user_name,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at,
    oauth_client_id,
    audience
FROM user_device_sessions
WHERE refresh_family = sqlc.arg(refresh_family)
  AND revoked_at IS NULL
LIMIT 1;
