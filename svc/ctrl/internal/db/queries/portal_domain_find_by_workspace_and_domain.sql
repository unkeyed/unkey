-- name: FindPortalDomainByWorkspaceAndDomain :one
-- Resolves a tenant's existing row for a hostname, letting the add path detect a
-- repeat request before it hits unique_workspace_domain_idx.
SELECT
    id,
    portal_id,
    domain,
    verification_status,
    invocation_id
FROM portal_domains
WHERE workspace_id = sqlc.arg(workspace_id) AND domain = sqlc.arg(domain);
