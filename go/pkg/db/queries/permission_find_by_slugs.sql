-- name: FindPermissionsBySlugs :many
SELECT * FROM permissions WHERE workspace_id = ? AND slug IN (sqlc.slice('slugs'));
