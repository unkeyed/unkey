-- name: ListPermissionsByRoleID :many
SELECT p.*
FROM permissions p
JOIN roles_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = sqlc.arg(role_id)
ORDER BY p.slug;
