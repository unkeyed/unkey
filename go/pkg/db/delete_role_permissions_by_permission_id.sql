-- name: DeleteRolePermissionsByPermissionId :execresult
-- Deletes all role-permission relationships for a specific permission
-- Returns: Result of the delete operation
DELETE FROM "role_permissions"
WHERE "permissionId" = $1;