-- name: CountPortalDomainsByWorkspace :one
-- Covered by unique_workspace_domain_idx, which leads on workspace_id.
SELECT COUNT(*)
FROM portal_domains
WHERE workspace_id = sqlc.arg(workspace_id);
