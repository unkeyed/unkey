-- name: ListDirectPermissionsByRoleID :many
SELECT p.id, p.name, p.slug, p.description
FROM roles_permissions rp
JOIN permissions p ON rp.permission_id = p.id
WHERE rp.role_id = sqlc.arg(role_id)
ORDER BY p.slug
FOR UPDATE;
