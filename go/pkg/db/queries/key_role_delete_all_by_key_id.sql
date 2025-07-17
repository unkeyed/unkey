-- name: DeleteAllKeyRolesByKeyID :exec
DELETE FROM keys_roles
WHERE key_id = sqlc.arg(key_id);