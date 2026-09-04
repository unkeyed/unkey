-- name: ListCustomDomains :many
-- ListCustomDomains applies optional resource filters cumulatively. Callers resolve
-- each supplied resource to its parent IDs before they run this query.
SELECT
    cd.id,
    cd.project_id,
    cd.app_id,
    cd.environment_id,
    cd.domain,
    cd.verification_status,
    cd.verification_token,
    cd.ownership_verified,
    cd.cname_verified,
    cd.target_cname,
    cd.verification_error,
    cd.domain_connect_provider,
    cd.domain_connect_url,
    cd.last_checked_at,
    cd.created_at,
    cd.updated_at
FROM custom_domains cd
WHERE cd.workspace_id = sqlc.arg(workspace_id)
  AND (sqlc.arg(project_id) = '' OR cd.project_id = sqlc.arg(project_id))
  AND (sqlc.arg(app_id) = '' OR cd.app_id = sqlc.arg(app_id))
  AND (sqlc.arg(environment_id) = '' OR cd.environment_id = sqlc.arg(environment_id))
  AND cd.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(cd.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd.id ASC
LIMIT ?;
