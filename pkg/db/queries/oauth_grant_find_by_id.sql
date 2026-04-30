-- name: FindOAuthGrantByID :one
SELECT id, workspace_id, provider, account_label, region, scopes,
       encrypted_credentials, encryption_key_id, expires_at, revoked_at,
       created_at, updated_at
FROM oauth_grants
WHERE id = ? AND revoked_at IS NULL;
