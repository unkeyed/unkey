-- name: DeleteAllKeyPermissionsByKeyID :exec
DELETE FROM keys_permissions
WHERE key_id = sqlc.arg(key_id);