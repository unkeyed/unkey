-- name: FindPermissionsByRoleId :many
-- Finds all permissions associated with a specific role
-- Returns: Permission records associated with the role
SELECT p.*
FROM "permissions" p
JOIN "role_permissions" rp ON p."id" = rp."permissionId"
WHERE rp."roleId" = $1
ORDER BY p."name";