-- name: FindRootKeyIPAllowlistEnabled :one
-- FindRootKeyIPAllowlistEnabled checks whether a workspace can configure root-key IP restrictions.
SELECT root_key_ip_allowlist_enabled FROM limits WHERE workspace_id = sqlc.arg(workspace_id);

-- name: UpdateRootKeyIPAllowlistEnabled :exec
-- UpdateRootKeyIPAllowlistEnabled changes workspace access to IP allowlist configuration.
-- Existing key restrictions remain enforced when configuration is disabled.
UPDATE limits SET root_key_ip_allowlist_enabled = sqlc.arg(enabled) WHERE workspace_id = sqlc.arg(workspace_id);
