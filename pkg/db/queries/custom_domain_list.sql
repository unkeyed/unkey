-- name: ListCustomDomains :many
-- ListCustomDomains filters by IDs resolved at the most specific supplied scope.
-- An empty scope lists the workspace. Callers authorize each returned domain.
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
  AND (cd.workspace_id = 'unkey_internal' OR (
    EXISTS (SELECT 1 FROM projects p WHERE p.id = cd.project_id AND p.deleted_at IS NULL)
    AND EXISTS (SELECT 1 FROM apps a WHERE a.id = cd.app_id AND a.deleted_at IS NULL)
    AND EXISTS (SELECT 1 FROM environments e WHERE e.id = cd.environment_id AND e.deleted_at IS NULL)
  ))
  AND (
    sqlc.arg(scope) = ''
    OR (sqlc.arg(scope) = 'project' AND cd.project_id IN (sqlc.slice(project_ids)))
    OR (sqlc.arg(scope) = 'app' AND cd.app_id IN (sqlc.slice(app_ids)))
    OR (sqlc.arg(scope) = 'environment' AND cd.environment_id IN (sqlc.slice(environment_ids)))
  )
  AND cd.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(cd.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd.id ASC
LIMIT ?;
