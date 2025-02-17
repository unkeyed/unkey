-- name: FindPermissionsForKey :many
WITH direct_permissions AS (
    SELECT p.name as permission_name
    FROM keys_permissions kp
    JOIN permissions p ON kp.permission_id = p.id
    WHERE kp.key_id = sqlc.arg(key_id)
),
role_permissions AS (
    SELECT p.name as permission_name
    FROM keys_roles kr
    JOIN roles_permissions rp ON kr.role_id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
    WHERE kr.key_id = sqlc.arg(key_id)
)
SELECT DISTINCT permission_name
FROM (
    SELECT permission_name FROM direct_permissions
    UNION ALL
    SELECT permission_name FROM role_permissions
) all_permissions;
