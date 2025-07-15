-- name: FindPermissionsBySlugs :many
SELECT id, slug FROM permissions WHERE workspace_id = ? AND slug IN (sqlc.slice('slugs'));
