-- name: ListEnvironmentsByApp :many
-- An app has only a handful of environments, so this returns all of them
-- without pagination.
SELECT environments.pk, environments.id, environments.workspace_id, environments.project_id, environments.app_id, environments.slug, environments.description, environments.kind, environments.delete_protection, environments.created_at, environments.updated_at
FROM environments
WHERE app_id = sqlc.arg(app_id)
ORDER BY id ASC;
