-- name: ListAppsByProject :many
SELECT
  apps.pk,
  apps.id,
  apps.workspace_id,
  apps.project_id,
  apps.name,
  apps.slug,
  apps.source_type,
  pe.current_deployment_id,
  pe.is_rolled_back,
  apps.delete_protection,
  apps.created_at,
  apps.updated_at,
  grc.repository_full_name AS repository_full_name,
  grc.default_branch AS github_default_branch,
  aso.image_reference AS oci_image_reference
FROM apps
LEFT JOIN github_repo_connections grc ON grc.app_id = apps.id
LEFT JOIN app_source_oci aso ON aso.app_id = apps.id
-- The app's oldest production environment is the one it was created with.
LEFT JOIN environments pe ON pe.id = (
  SELECT e.id
  FROM environments e
  WHERE e.app_id = apps.id
    AND e.kind = 'production'
    AND e.deleted_at IS NULL
  ORDER BY e.pk ASC
  LIMIT 1
)
WHERE apps.project_id = sqlc.arg(project_id)
  AND apps.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM projects p WHERE p.id = apps.project_id AND p.deleted_at IS NULL)
  AND apps.id >= sqlc.arg(id_cursor)
  -- search is a pre-escaped LIKE pattern built by mysql.SearchContains; NULL disables the filter
  AND (sqlc.narg(search) IS NULL OR LOWER(apps.id) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.name) LIKE LOWER(sqlc.narg(search)) OR LOWER(apps.slug) LIKE LOWER(sqlc.narg(search)))
ORDER BY apps.id ASC
LIMIT ?;
