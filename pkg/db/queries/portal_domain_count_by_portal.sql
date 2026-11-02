-- name: CountPortalDomainsByPortal :one
-- Counts a portal's domains so deletePortal only asks ctrl to delete them when
-- there are any.
SELECT COUNT(*)
FROM portal_domains
WHERE portal_id = sqlc.arg(portal_id)
  AND workspace_id = sqlc.arg(workspace_id);
