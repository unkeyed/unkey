-- name: DeleteManyKeyRolesByKeyAndRoleIDs :exec
DELETE FROM keys_roles
WHERE key_id = sqlc.arg('key_id') AND role_id IN(sqlc.slice('role_ids'));
