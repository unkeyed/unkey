-- name: DeleteManyKeyRolesByRoleID :exec
DELETE FROM keys_roles
WHERE role_id = sqlc.arg(role_id);