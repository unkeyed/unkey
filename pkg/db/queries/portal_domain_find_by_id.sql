-- name: FindPortalDomainById :one
-- Scoped to the workspace so a caller cannot read another tenant's domain by
-- guessing its id. Callers compare portal_id with the portal they resolved,
-- which lets this run concurrently with the portal lookup.
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
LIMIT 1;
