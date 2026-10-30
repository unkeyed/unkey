-- name: FindIPAllowlistEnabled :one
-- FindIPAllowlistEnabled checks whether a workspace can configure IP restrictions.
SELECT ip_allowlist_enabled FROM limits WHERE workspace_id = sqlc.arg(workspace_id);

-- name: UpdateIPAllowlistEnabled :exec
-- UpdateIPAllowlistEnabled changes workspace access to IP allowlist configuration.
-- Existing key restrictions remain enforced when configuration is disabled.
UPDATE limits SET ip_allowlist_enabled = sqlc.arg(enabled) WHERE workspace_id = sqlc.arg(workspace_id);
