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
-- Names are stored lowercase, so the name half is lowered here rather than left to the
-- column's collation. The id half is compared as given: ids are case-sensitive.
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (id = sqlc.arg(domain) OR domain = LOWER(sqlc.arg(domain)))
LIMIT 1;
