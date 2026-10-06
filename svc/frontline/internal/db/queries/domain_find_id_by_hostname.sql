-- name: FindDomainIDByHostname :one
-- Checks whether a hostname is registered as a deploy or portal custom domain.
-- ACME HTTP-01 only needs to confirm a registration exists, so the projection is
-- just a row id.
SELECT custom_domains.id FROM custom_domains WHERE custom_domains.domain = sqlc.arg(domain)
UNION ALL
SELECT portal_domains.id FROM portal_domains WHERE portal_domains.domain = sqlc.arg(domain)
LIMIT 1;
