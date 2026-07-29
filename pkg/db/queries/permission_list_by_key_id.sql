-- name: ListPermissionsByKeyID :many
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
WITH direct_permissions AS (
    SELECT p.slug as permission_slug
    FROM keys_permissions kp
    JOIN permissions p ON (kp.permission_id = p.id COLLATE utf8mb4_0900_ai_ci AND kp.permission_id = p.id COLLATE utf8mb4_0900_as_cs)
    WHERE kp.key_id = sqlc.arg(key_id)
),
role_permissions AS (
    SELECT p.slug as permission_slug
    FROM keys_roles kr
    JOIN roles_permissions rp ON (kr.role_id = rp.role_id COLLATE utf8mb4_0900_ai_ci AND kr.role_id = rp.role_id COLLATE utf8mb4_0900_as_cs)
    JOIN permissions p ON (rp.permission_id = p.id COLLATE utf8mb4_0900_ai_ci AND rp.permission_id = p.id COLLATE utf8mb4_0900_as_cs)
    WHERE kr.key_id = sqlc.arg(key_id)
)
SELECT DISTINCT permission_slug
FROM (
    SELECT permission_slug FROM direct_permissions
    UNION ALL
    SELECT permission_slug FROM role_permissions
) all_permissions;
