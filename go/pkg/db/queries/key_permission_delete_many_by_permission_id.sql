-- name: DeleteManyKeyPermissionsByPermissionID :exec
DELETE FROM keys_permissions
WHERE permission_id = sqlc.Arg(permission_id);
