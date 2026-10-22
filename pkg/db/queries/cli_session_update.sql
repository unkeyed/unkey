-- name: RotateCLISessionRefreshToken :execrows
UPDATE cli_sessions
SET
    refresh_token_hash = sqlc.arg(refresh_token_hash),
    previous_refresh_token_hash = sqlc.arg(previous_refresh_token_hash),
    rotated_at = sqlc.arg(rotated_at),
    last_used_at = sqlc.arg(last_used_at),
    expires_at = sqlc.arg(expires_at),
    permissions = CAST(sqlc.arg(permissions) AS JSON)
WHERE id = sqlc.arg(id)
  AND refresh_token_hash = sqlc.arg(current_refresh_token_hash)
  AND revoked_at IS NULL;

-- name: RevokeCLISession :execrows
UPDATE cli_sessions
SET revoked_at = sqlc.arg(revoked_at)
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND revoked_at IS NULL;

-- name: RevokeCLISessionFamily :execrows
UPDATE cli_sessions
SET revoked_at = sqlc.arg(revoked_at)
WHERE refresh_family = sqlc.arg(refresh_family)
  AND revoked_at IS NULL;

-- name: DeleteExpiredCLISessions :execrows
DELETE FROM cli_sessions
WHERE expires_at < sqlc.arg(expired_before)
LIMIT 100;
