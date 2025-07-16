-- name: FindPermissionsBySlugs :many
SELECT id, slug, name, description FROM permissions WHERE workspace_id = ? AND slug IN (sqlc.slice('slugs'));
