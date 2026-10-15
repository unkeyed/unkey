-- name: FindVerifiedDomainByHostname :one
-- Resolves a hostname to its verified row in custom_domains or portal_domains so
-- certificate issuance never acts on a pending competing claim. Verification
-- revokes contested claims first; if two verified rows ever coexist, the most
-- recently changed one wins.
SELECT d.id, d.workspace_id, d.domain
FROM (
    SELECT custom_domains.id, custom_domains.workspace_id, custom_domains.domain,
        COALESCE(custom_domains.updated_at, custom_domains.created_at) AS changed_at
    FROM custom_domains
    WHERE custom_domains.domain = sqlc.arg(domain)
      AND custom_domains.verification_status = 'verified'
    UNION ALL
    SELECT portal_domains.id, portal_domains.workspace_id, portal_domains.domain,
        COALESCE(portal_domains.updated_at, portal_domains.created_at) AS changed_at
    FROM portal_domains
    WHERE portal_domains.domain = sqlc.arg(domain)
      AND portal_domains.verification_status = 'verified'
) AS d
ORDER BY d.changed_at DESC, d.id DESC
LIMIT 1;
