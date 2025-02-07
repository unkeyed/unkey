-- name: FindKeyForVerification :one
WITH direct_permissions AS (
    SELECT kp.key_id, p.name as permission_name
    FROM keys_permissions kp
    JOIN permissions p ON kp.permission_id = p.id
),
role_permissions AS (
    SELECT kr.key_id, p.name as permission_name
    FROM keys_roles kr
    JOIN roles_permissions rp ON kr.role_id = rp.role_id
    JOIN permissions p ON rp.permission_id = p.id
),
all_permissions AS (
    SELECT * FROM direct_permissions
    UNION
    SELECT * FROM role_permissions
),
all_ratelimits AS (
    SELECT
        key_id as target_id,
        'key' as target_type,
        name,
        `limit`,
        duration
    FROM ratelimits
    WHERE key_id IS NOT NULL
    UNION
    SELECT
        identity_id as target_id,
        'identity' as target_type,
        name,
        `limit`,
        duration
    FROM ratelimits
    WHERE identity_id IS NOT NULL
)
SELECT
    sqlc.embed(k),
    sqlc.embed(i),
    JSON_ARRAYAGG(
        JSON_OBJECT(
            'target_type', rl.target_type,
            'name', rl.name,
            'limit', rl.limit,
            'duration', rl.duration
        )
    ) as ratelimits,
    GROUP_CONCAT(DISTINCT perms.permission_name) as permissions
FROM `keys` k
LEFT JOIN identities i ON k.identity_id = i.id
LEFT JOIN all_permissions perms ON k.id = perms.key_id
LEFT JOIN all_ratelimits rl ON (
    (rl.target_type = 'key' AND rl.target_id = k.id) OR
    (rl.target_type = 'identity' AND rl.target_id = k.identity_id)
)
WHERE k.hash = ?
GROUP BY k.id;
