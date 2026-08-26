-- name: ListCustomDomains :many
-- ListCustomDomains uses one mutually exclusive branch for each resolved scope.
-- UNION ALL lets MySQL select the matching workspace, project, app, or environment
-- index instead of combining the optional filters with index-blocking OR predicates.
(SELECT
    cd_workspace.id,
    cd_workspace.project_id,
    cd_workspace.app_id,
    cd_workspace.environment_id,
    cd_workspace.domain,
    cd_workspace.verification_status,
    cd_workspace.verification_token,
    cd_workspace.ownership_verified,
    cd_workspace.cname_verified,
    cd_workspace.target_cname,
    cd_workspace.verification_error,
    cd_workspace.domain_connect_provider,
    cd_workspace.domain_connect_url,
    cd_workspace.last_checked_at,
    cd_workspace.created_at,
    cd_workspace.updated_at
FROM custom_domains cd_workspace
WHERE sqlc.arg(project_id) = ''
  AND sqlc.arg(app_id) = ''
  AND sqlc.arg(environment_id) = ''
  AND cd_workspace.workspace_id = sqlc.arg(workspace_id)
  AND cd_workspace.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(cd_workspace.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd_workspace.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd_workspace.id ASC
LIMIT ?)
UNION ALL
(SELECT
    cd_project.id,
    cd_project.project_id,
    cd_project.app_id,
    cd_project.environment_id,
    cd_project.domain,
    cd_project.verification_status,
    cd_project.verification_token,
    cd_project.ownership_verified,
    cd_project.cname_verified,
    cd_project.target_cname,
    cd_project.verification_error,
    cd_project.domain_connect_provider,
    cd_project.domain_connect_url,
    cd_project.last_checked_at,
    cd_project.created_at,
    cd_project.updated_at
FROM custom_domains cd_project
WHERE sqlc.arg(project_id) != ''
  AND sqlc.arg(app_id) = ''
  AND sqlc.arg(environment_id) = ''
  AND cd_project.workspace_id = sqlc.arg(workspace_id)
  AND cd_project.project_id = sqlc.arg(project_id)
  AND cd_project.id >= sqlc.arg(id_cursor)
  AND (sqlc.narg(search) IS NULL OR LOWER(cd_project.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd_project.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd_project.id ASC
LIMIT ?)
UNION ALL
(SELECT
    cd_app.id,
    cd_app.project_id,
    cd_app.app_id,
    cd_app.environment_id,
    cd_app.domain,
    cd_app.verification_status,
    cd_app.verification_token,
    cd_app.ownership_verified,
    cd_app.cname_verified,
    cd_app.target_cname,
    cd_app.verification_error,
    cd_app.domain_connect_provider,
    cd_app.domain_connect_url,
    cd_app.last_checked_at,
    cd_app.created_at,
    cd_app.updated_at
FROM custom_domains cd_app
WHERE sqlc.arg(app_id) != ''
  AND sqlc.arg(environment_id) = ''
  AND cd_app.workspace_id = sqlc.arg(workspace_id)
  AND cd_app.app_id = sqlc.arg(app_id)
  AND cd_app.id >= sqlc.arg(id_cursor)
  AND (sqlc.narg(search) IS NULL OR LOWER(cd_app.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd_app.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd_app.id ASC
LIMIT ?)
UNION ALL
(SELECT
    cd_environment.id,
    cd_environment.project_id,
    cd_environment.app_id,
    cd_environment.environment_id,
    cd_environment.domain,
    cd_environment.verification_status,
    cd_environment.verification_token,
    cd_environment.ownership_verified,
    cd_environment.cname_verified,
    cd_environment.target_cname,
    cd_environment.verification_error,
    cd_environment.domain_connect_provider,
    cd_environment.domain_connect_url,
    cd_environment.last_checked_at,
    cd_environment.created_at,
    cd_environment.updated_at
FROM custom_domains cd_environment
WHERE sqlc.arg(environment_id) != ''
  AND cd_environment.workspace_id = sqlc.arg(workspace_id)
  AND cd_environment.environment_id = sqlc.arg(environment_id)
  AND cd_environment.id >= sqlc.arg(id_cursor)
  AND (sqlc.narg(search) IS NULL OR LOWER(cd_environment.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd_environment.domain) LIKE LOWER(sqlc.narg(search)))
ORDER BY cd_environment.id ASC
LIMIT ?)
ORDER BY id ASC
LIMIT ?;
