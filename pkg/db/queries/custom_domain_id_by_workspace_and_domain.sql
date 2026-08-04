-- name: FindCustomDomainIDByWorkspaceAndDomain :one
-- Covered by unique_domain_workspace_idx.
SELECT id
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id)
  AND domain = sqlc.arg(domain)
LIMIT 1;
