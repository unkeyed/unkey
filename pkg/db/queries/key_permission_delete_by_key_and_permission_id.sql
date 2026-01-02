-- name: DeleteKeyPermissionByKeyAndPermissionID :exec
DELETE FROM keys_permissions
WHERE key_id = sqlc.arg(key_id) AND permission_id = sqlc.arg(permission_id);