-- name: FindKeyRoleByKeyAndRoleID :many
SELECT keys_roles.pk, keys_roles.key_id, keys_roles.role_id, keys_roles.workspace_id, keys_roles.created_at_m, keys_roles.updated_at_m
FROM keys_roles
WHERE key_id = sqlc.arg(key_id)
  AND role_id = sqlc.arg(role_id);
