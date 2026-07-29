-- name: FindLiveKeyByID :one
-- Temporary staged-collation bridge: the native-collation term preserves
-- index lookup while the as_cs term enforces exact ID equality. Remove after
-- all counterpart columns are utf8mb4_0900_as_cs.
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
        WHERE (k.id COLLATE utf8mb4_0900_ai_ci = kr.key_id AND k.id COLLATE utf8mb4_0900_as_cs = kr.key_id)),
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
        WHERE (k.id COLLATE utf8mb4_0900_ai_ci = kp.key_id AND k.id COLLATE utf8mb4_0900_as_cs = kp.key_id)),
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
        WHERE (k.id COLLATE utf8mb4_0900_ai_ci = kr.key_id AND k.id COLLATE utf8mb4_0900_as_cs = kr.key_id)),
        JSON_ARRAY()
    ) as role_permissions,

    -- Rate limits
    COALESCE(
        (SELECT JSON_ARRAYAGG(
            JSON_OBJECT(
                'id', rl.id,
                'name', rl.name,
                'key_id', rl.key_id,
                'identity_id', rl.identity_id,
                'limit', rl.`limit`,
                'duration', rl.duration,
                'auto_apply', rl.auto_apply = 1
            )
        )
        FROM ratelimits rl
        WHERE (k.id COLLATE utf8mb4_0900_ai_ci = rl.key_id AND k.id COLLATE utf8mb4_0900_as_cs = rl.key_id)
            OR (i.id COLLATE utf8mb4_0900_ai_ci = rl.identity_id AND i.id COLLATE utf8mb4_0900_as_cs = rl.identity_id)),
        JSON_ARRAY()
    ) as ratelimits

FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON (k.workspace_id COLLATE utf8mb4_0900_ai_ci = ws.id AND k.workspace_id COLLATE utf8mb4_0900_as_cs = ws.id)
LEFT JOIN identities i ON (k.identity_id COLLATE utf8mb4_0900_ai_ci = i.id AND k.identity_id COLLATE utf8mb4_0900_as_cs = i.id) AND i.deleted = false
LEFT JOIN encrypted_keys ek ON (k.id COLLATE utf8mb4_0900_ai_ci = ek.key_id AND k.id COLLATE utf8mb4_0900_as_cs = ek.key_id)
WHERE k.id = sqlc.arg(id)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;
