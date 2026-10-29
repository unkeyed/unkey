-- name: ListPortalDomainsByPortal :many
-- Lists every domain attached to a portal, verified or not, in id order.
-- Workspace-scoped so a portal id from another tenant returns nothing.
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
WHERE portal_id = sqlc.arg(portal_id)
  AND workspace_id = sqlc.arg(workspace_id)
ORDER BY id ASC;
