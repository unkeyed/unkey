-- name: ListEnvironmentsByApp :many
-- An app has only a handful of environments, so this returns all of them
-- without pagination.
SELECT environments.pk, environments.id, environments.workspace_id, environments.project_id, environments.app_id, environments.slug, environments.description, environments.kind, environments.delete_protection, environments.created_at, environments.updated_at, environments.deleted_at
FROM environments
WHERE app_id = sqlc.arg(app_id)
  AND environments.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM apps a JOIN projects p ON p.id = a.project_id
    WHERE a.id = environments.app_id AND a.deleted_at IS NULL AND p.deleted_at IS NULL
  )
ORDER BY id ASC;
