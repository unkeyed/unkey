-- name: FindLiveKeyByID :one
SELECT
    k.*,
    sqlc.embed(a),
    sqlc.embed(ka),
    sqlc.embed(ws),

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
        WHERE kr.key_id = k.id),
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
        WHERE kp.key_id = k.id),
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
        WHERE kr.key_id = k.id),
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
        WHERE rl.key_id = k.id
            OR rl.identity_id = i.id),
        JSON_ARRAY()
    ) as ratelimits,

    -- Identity
    i.id as identity_table_id,
    i.external_id as identity_external_id,
    i.meta as identity_meta,

    -- Credits Key/Identity based
    kc.id as credit_id,
    kc.remaining as credit_remaining,
    kc.refill_amount as credit_refill_amount,
    kc.refill_day as credit_refill_day,
    kc.refilled_at as credit_refilled_at,

    ic.id as identity_credit_id,
    ic.remaining as identity_credit_remaining,
    ic.refill_amount as identity_credit_refill_amount,
    ic.refill_day as identity_credit_refill_day,
    ic.refilled_at as identity_credit_refilled_at

FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
LEFT JOIN encrypted_keys ek ON ek.key_id = k.id
LEFT JOIN credits kc ON kc.key_id = k.id
LEFT JOIN credits ic ON ic.identity_id = i.id
WHERE k.id = sqlc.arg(id)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;
