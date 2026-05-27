-- name: FindCustomDomainIDByDomain :one
-- FindCustomDomainIDByDomain checks whether the caller-supplied domain has been
-- registered as a custom domain. ACME HTTP-01 only needs to confirm ownership,
-- not resolve any routing data, so the projection is just the row id.
SELECT id FROM custom_domains WHERE domain = sqlc.arg(domain);
