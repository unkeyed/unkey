-- name: FindCustomDomainByDomain :one
SELECT
    id,
    workspace_id,
    domain,
    challenge_type,
    created_at,
    updated_at
FROM custom_domains
WHERE domain = sqlc.arg(domain);
