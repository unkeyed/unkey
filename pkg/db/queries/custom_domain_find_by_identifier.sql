-- name: FindCustomDomainByIdentifier :one
SELECT
    id,
    project_id,
    app_id,
    environment_id,
    domain,
    verification_status,
    verification_token,
    ownership_verified,
    cname_verified,
    target_cname,
    verification_error,
    last_checked_at,
    created_at,
    updated_at
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg(domain) OR domain = sqlc.arg(domain))
LIMIT 1;
