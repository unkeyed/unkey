-- name: ListCustomDomainsByEnvironment :many
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
WHERE project_id = sqlc.arg(project_id)
  AND environment_id = sqlc.arg(environment_id)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(id) LIKE LOWER(sqlc.narg(search)) OR LOWER(domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY id ASC
LIMIT ?;
