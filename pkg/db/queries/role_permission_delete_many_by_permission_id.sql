-- name: DeleteManyRolePermissionsByPermissionID :exec
DELETE FROM roles_permissions
WHERE permission_id = sqlc.arg(permission_id);
