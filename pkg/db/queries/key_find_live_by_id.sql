-- name: FindLiveKeyByID :one
SELECT
    k.id AS key_id,
    k.key_auth_id AS key_key_auth_id,
    k.hash AS key_hash,
    k.start AS key_start,
    k.workspace_id AS key_workspace_id,
    k.for_workspace_id AS key_for_workspace_id,
    k.name AS key_name,
    k.identity_id AS key_identity_id,
    k.meta AS key_meta,
    k.expires AS key_expires,
    k.created_at_m AS key_created_at_m,
    k.updated_at_m AS key_updated_at_m,
    k.refill_day AS key_refill_day,
    k.refill_amount AS key_refill_amount,
    k.enabled AS key_enabled,
    k.remaining_requests AS key_remaining_requests,
    k.last_used_at AS key_last_used_at,
    a.id AS api_id,
    a.name AS api_name,
    ka.id AS key_auth_id,
    ka.store_encrypted_keys AS key_auth_store_encrypted_keys,
    ka.default_prefix AS key_auth_default_prefix,
    ka.default_bytes AS key_auth_default_bytes,
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
        WHERE k.id = rl.key_id
            OR i.id = rl.identity_id),
        JSON_ARRAY()
    ) as ratelimits

FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON k.workspace_id = ws.id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
LEFT JOIN encrypted_keys ek ON k.id = ek.key_id
WHERE k.id = sqlc.arg(id)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;
