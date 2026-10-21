-- name: CountPortalDomainsByPortal :one
-- Counts a portal's domains so deletePortal can refuse while any remain, since
-- a deleted portal would otherwise leave them routed and unmanageable.
SELECT COUNT(*)
FROM portal_domains
WHERE portal_id = sqlc.arg(portal_id)
  AND workspace_id = sqlc.arg(workspace_id);
