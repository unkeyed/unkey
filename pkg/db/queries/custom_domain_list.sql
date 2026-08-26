-- name: ListCustomDomains :many
-- ListCustomDomains applies optional hierarchy filters after the handler resolves
-- identifiers to IDs. An empty filter includes every child scope in the workspace.
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
    domain_connect_provider,
    domain_connect_url,
    last_checked_at,
    created_at,
    updated_at
FROM custom_domains
WHERE workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR environment_id = sqlc.arg(environment_id))
  AND id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(id) LIKE LOWER(sqlc.narg(search)) OR LOWER(domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY id ASC
LIMIT ?;
