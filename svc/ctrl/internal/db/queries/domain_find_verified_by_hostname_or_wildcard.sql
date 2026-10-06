-- name: FindVerifiedDomainByHostnameOrWildcard :one
-- Resolves an ACME challenge hostname to a verified domain row, preferring an
-- exact match in either domain table over a custom_domains wildcard. Portal
-- domains are never wildcards, so only custom_domains is checked for one.
SELECT d.id, d.workspace_id, d.domain
FROM (
    SELECT custom_domains.id, custom_domains.workspace_id, custom_domains.domain, 0 AS is_wildcard,
        COALESCE(custom_domains.updated_at, custom_domains.created_at) AS changed_at
    FROM custom_domains
    WHERE custom_domains.domain = sqlc.arg(domain)
      AND custom_domains.verification_status = 'verified'
    UNION ALL
    SELECT portal_domains.id, portal_domains.workspace_id, portal_domains.domain, 0 AS is_wildcard,
        COALESCE(portal_domains.updated_at, portal_domains.created_at) AS changed_at
    FROM portal_domains
    WHERE portal_domains.domain = sqlc.arg(domain)
      AND portal_domains.verification_status = 'verified'
    UNION ALL
    SELECT custom_domains.id, custom_domains.workspace_id, custom_domains.domain, 1 AS is_wildcard,
        COALESCE(custom_domains.updated_at, custom_domains.created_at) AS changed_at
    FROM custom_domains
    WHERE custom_domains.domain = sqlc.arg(wildcard_domain)
      AND custom_domains.verification_status = 'verified'
) AS d
ORDER BY d.is_wildcard ASC, d.changed_at DESC, d.id DESC
LIMIT 1;
