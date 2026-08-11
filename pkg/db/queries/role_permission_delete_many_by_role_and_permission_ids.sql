-- name: DeleteManyRolePermissionsByRoleAndPermissionIDs :exec
DELETE FROM roles_permissions
WHERE role_id = sqlc.arg(role_id) AND permission_id IN (sqlc.slice(permission_ids));
