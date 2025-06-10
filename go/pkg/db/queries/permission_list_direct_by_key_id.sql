-- name: ListDirectPermissionsByKeyID :many
SELECT p.id, p.workspace_id, p.name, p.slug, p.description, p.created_at_m, p.updated_at_m
FROM keys_permissions kp
JOIN permissions p ON kp.permission_id = p.id
WHERE kp.key_id = sqlc.arg(key_id)
ORDER BY p.name;