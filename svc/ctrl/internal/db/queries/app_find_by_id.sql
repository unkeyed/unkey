-- name: FindAppById :one
SELECT apps.pk, apps.id, apps.workspace_id, apps.project_id, apps.name, apps.slug, apps.source_type, apps.current_deployment_id, apps.is_rolled_back, apps.delete_protection, apps.created_at, apps.updated_at
FROM apps
WHERE id = sqlc.arg(id);
