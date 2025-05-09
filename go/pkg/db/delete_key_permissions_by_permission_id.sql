-- name: DeleteKeyPermissionsByPermissionId :execresult
-- Deletes all key-permission relationships for a specific permission
-- Returns: Result of the delete operation
DELETE FROM "key_permissions"
WHERE "permissionId" = $1;