-- name: FindPortalDomainById :one
-- Scoped to workspace and portal so a caller cannot read a domain attached to
-- another tenant's portal by guessing its id.
SELECT
    id,
    workspace_id,
    portal_id,
    domain,
    verification_status,
    verification_token,
    ownership_verified,
    cname_verified,
    target_cname,
    verification_error,
    domain_connect_provider,
    domain_connect_url,
    last_checked_at,
    created_at,
    updated_at
FROM portal_domains
WHERE id = sqlc.arg(id)
  AND workspace_id = sqlc.arg(workspace_id)
  AND portal_id = sqlc.arg(portal_id)
LIMIT 1;
