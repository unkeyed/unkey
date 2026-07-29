-- name: ListDirectPermissionsByKeyID :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
SELECT p.*
FROM keys_permissions kp
JOIN permissions p ON (kp.permission_id = p.id COLLATE utf8mb4_0900_ai_ci AND kp.permission_id = p.id COLLATE utf8mb4_0900_as_cs)
WHERE kp.key_id = sqlc.arg(key_id)
ORDER BY p.slug;
