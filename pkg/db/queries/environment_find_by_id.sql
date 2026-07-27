-- name: FindEnvironmentById :one
SELECT environments.pk, environments.id, environments.workspace_id, environments.project_id, environments.app_id, environments.slug, environments.description, environments.delete_protection, environments.created_at, environments.updated_at
FROM environments
WHERE id = sqlc.arg(id);
