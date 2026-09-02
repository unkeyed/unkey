-- name: FindPermissionsBySlugs :many
SELECT permissions.pk, permissions.id, permissions.workspace_id, permissions.project_id, permissions.name, permissions.slug, permissions.description, permissions.created_at_m, permissions.updated_at_m FROM permissions WHERE workspace_id = ? AND slug IN (sqlc.slice('slugs'));
