-- name: DeleteManyKeyPermissionByKeyAndPermissionIDs :exec
DELETE FROM keys_permissions
WHERE key_id = sqlc.arg(key_id) AND permission_id IN (sqlc.slice(ids));
