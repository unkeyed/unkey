-- name: ListCustomDomains :many
-- ListCustomDomains selects IDs through one mutually exclusive scope branch, then
-- loads the result rows once. Each branch is covered by its scope, ID, and domain
-- index, so UNION ALL avoids the index-blocking optional OR predicates.
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
JOIN (
    (SELECT cd_workspace.id
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
    (SELECT cd_project.id
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
    (SELECT cd_app.id
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
    (SELECT cd_environment.id
    FROM custom_domains cd_environment
    WHERE sqlc.arg(environment_id) != ''
      AND cd_environment.workspace_id = sqlc.arg(workspace_id)
      AND cd_environment.environment_id = sqlc.arg(environment_id)
      AND cd_environment.id >= sqlc.arg(id_cursor)
      AND (sqlc.narg(search) IS NULL OR LOWER(cd_environment.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(cd_environment.domain) LIKE LOWER(sqlc.narg(search)))
    ORDER BY cd_environment.id ASC
    LIMIT ?)
) scoped_domains ON scoped_domains.id = cd.id
ORDER BY cd.id ASC;
