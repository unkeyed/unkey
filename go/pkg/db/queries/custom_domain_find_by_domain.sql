-- name: FindCustomDomainByDomain :one
SELECT *
FROM custom_domains
WHERE domain = sqlc.arg(domain);
