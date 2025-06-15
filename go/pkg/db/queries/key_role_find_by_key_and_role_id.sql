-- name: FindKeyRoleByKeyAndRoleID :many
SELECT *
FROM keys_roles
WHERE key_id = sqlc.arg(key_id)
  AND role_id = sqlc.arg(role_id);