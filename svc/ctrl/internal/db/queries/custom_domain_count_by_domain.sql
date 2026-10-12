-- name: CountCustomDomainsByDomain :one
-- CountCustomDomainsByDomain counts custom domain rows for a hostname across
-- all workspaces. Only tests use this query.
SELECT COUNT(*)
FROM `custom_domains`
WHERE domain = sqlc.arg(domain);
