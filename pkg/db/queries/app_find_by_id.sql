-- name: FindAppById :one
SELECT apps.pk, apps.id, apps.workspace_id, apps.project_id, apps.name, apps.slug, apps.source_type, apps.delete_protection, apps.created_at, apps.updated_at, apps.deleted_at
FROM apps
WHERE id = sqlc.arg(id);
