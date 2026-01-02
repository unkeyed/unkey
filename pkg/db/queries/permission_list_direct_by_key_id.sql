-- name: ListDirectPermissionsByKeyID :many
SELECT p.*
FROM keys_permissions kp
JOIN permissions p ON kp.permission_id = p.id
WHERE kp.key_id = sqlc.arg(key_id)
ORDER BY p.slug;
