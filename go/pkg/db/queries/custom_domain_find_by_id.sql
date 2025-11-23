-- name: FindCustomDomainById :one
SELECT
    id,
    workspace_id,
    domain,
    created_at,
    updated_at
FROM custom_domains
WHERE id = sqlc.arg(id);
