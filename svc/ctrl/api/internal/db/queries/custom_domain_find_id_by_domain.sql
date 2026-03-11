-- name: FindCustomDomainIDByDomain :one
-- FindCustomDomainIDByDomain checks whether a custom domain row already exists
-- for the provided fully qualified domain name.
--
-- We select only the primary identifier instead of the full row because the
-- caller only needs existence semantics before deciding whether to create
-- bootstrap records.
SELECT
  id
FROM custom_domains
WHERE domain = sqlc.arg(domain);
