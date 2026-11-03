-- name: FindDomainIDByHostname :one
-- Checks whether a hostname is a verified deploy or portal custom domain.
-- Challenges are only issued for verified domains, so pending claims on the same
-- hostname are ignored. The projection is just a row id.
SELECT custom_domains.id FROM custom_domains
WHERE custom_domains.domain = sqlc.arg(domain)
  AND custom_domains.verification_status = 'verified'
UNION ALL
SELECT portal_domains.id FROM portal_domains
WHERE portal_domains.domain = sqlc.arg(domain)
  AND portal_domains.verification_status = 'verified'
LIMIT 1;
