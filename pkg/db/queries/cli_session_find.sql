-- name: FindCLISessionByID :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM cli_sessions
WHERE id = sqlc.arg(id);

-- name: FindCLISessionByRefreshTokenHash :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM cli_sessions
WHERE refresh_token_hash = sqlc.arg(refresh_token_hash);

-- name: FindCLISessionByPreviousRefreshTokenHash :one
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM cli_sessions
WHERE previous_refresh_token_hash = sqlc.arg(previous_refresh_token_hash);

-- name: ListCLISessionsByWorkspace :many
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM cli_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;

-- name: ListCLISessionsByUser :many
SELECT
    pk,
    id,
    workspace_id,
    user_id,
    permissions,
    device_name,
    refresh_token_hash,
    previous_refresh_token_hash,
    refresh_family,
    rotated_at,
    created_at,
    last_used_at,
    expires_at,
    revoked_at
FROM cli_sessions
WHERE workspace_id = sqlc.arg(workspace_id)
  AND user_id = sqlc.arg(user_id)
  AND revoked_at IS NULL
  AND expires_at > sqlc.arg(now)
ORDER BY created_at DESC
LIMIT 100;
