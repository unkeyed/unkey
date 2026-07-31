-- name: FindLiveKeyByHash :one
SELECT
    k.*,
    sqlc.embed(a),
    sqlc.embed(ka),
    sqlc.embed(ws),
    i.id as identity_table_id,
    i.external_id as identity_external_id,
    i.meta as identity_meta,
    ek.encrypted as encrypted_key,
    ek.encryption_key_id as encryption_key_id,

    -- Roles with both IDs and names
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', r.id,
                'name', r.name,
                'description', r.description
            )
        )
        FROM keys_roles kr
        JOIN roles r ON r.id = kr.role_id
        WHERE k.id = kr.key_id),
        JSON_ARRAY()
    ) as roles,

    -- Direct permissions attached to the key
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_permissions kp
        JOIN permissions p ON kp.permission_id = p.id
        WHERE k.id = kp.key_id),
        JSON_ARRAY()
    ) as permissions,

    -- Permissions from roles
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', p.id,
                'name', p.name,
                'slug', p.slug,
                'description', p.description
            )
        )
        FROM keys_roles kr
        JOIN roles_permissions rp ON kr.role_id = rp.role_id
        JOIN permissions p ON rp.permission_id = p.id
        WHERE k.id = kr.key_id),
        JSON_ARRAY()
    ) as role_permissions,

    -- Rate limits
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', id,
                'name', name,
                'key_id', key_id,
                'identity_id', identity_id,
                'limit', `limit`,
                'duration', duration,
                'auto_apply', auto_apply = 1
            )
        )
        FROM (
            SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
            FROM ratelimits rl
            WHERE k.id = rl.key_id
            UNION ALL
            SELECT rl.id, rl.name, rl.key_id, rl.identity_id, rl.`limit`, rl.duration, rl.auto_apply
            FROM ratelimits rl
            WHERE i.id = rl.identity_id
        ) AS combined_rl),
        JSON_ARRAY()
    ) as ratelimits

FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON k.workspace_id = ws.id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.hash = sqlc.arg(hash)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;
