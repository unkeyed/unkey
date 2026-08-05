-- name: FindCustomDomainById :one
SELECT
    id,
    domain,
    verification_token,
    target_cname
FROM custom_domains
WHERE id = sqlc.arg(id)
LIMIT 1;
