-- name: ListPreviewEnvironments :many
SELECT environments.pk, environments.id, environments.workspace_id, environments.project_id, environments.app_id, environments.slug, environments.description, environments.delete_protection, environments.created_at, environments.updated_at
FROM environments
WHERE slug = 'preview'
AND pk > sqlc.arg(pagination_cursor)
ORDER BY pk ASC
LIMIT ?;
