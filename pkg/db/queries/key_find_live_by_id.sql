-- name: FindLiveKeyByID :one
SELECT
    k.pk, k.id, k.key_auth_id, k.hash, k.start, k.workspace_id, k.for_workspace_id,
    k.name, k.identity_id, k.meta, k.expires, k.created_at_m, k.updated_at_m,
    k.deleted_at_m, k.refill_day, k.refill_amount, k.last_refill_at, k.enabled,
    k.remaining_requests, k.environment, k.last_used_at, k.pending_migration_id,
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

-- name: FindKeyMutationResources :many
-- Resolve optional identity, permission, role, and project references in one round trip.
SELECT
    'identity' AS resource_type,
    i.id AS identity_id,
    i.external_id AS identity_external_id,
    '' AS permission_id,
    '' AS permission_slug,
    '' AS role_id,
    '' AS role_name,
    '' AS project_id,
    '' AS project_slug
FROM identities i
WHERE CAST(sqlc.arg(find_identity) AS UNSIGNED) = 1
    AND i.workspace_id = sqlc.arg(workspace_id)
    AND i.external_id = sqlc.arg(external_id)
    AND i.deleted = false
UNION ALL
SELECT
    'permission' AS resource_type,
    '' AS identity_id,
    '' AS identity_external_id,
    p.id AS permission_id,
    p.slug AS permission_slug,
    '' AS role_id,
    '' AS role_name,
    '' AS project_id,
    '' AS project_slug
FROM permissions p
WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND p.slug IN (sqlc.slice('permission_slugs'))
UNION ALL
SELECT
    'role' AS resource_type,
    '' AS identity_id,
    '' AS identity_external_id,
    '' AS permission_id,
    '' AS permission_slug,
    r.id AS role_id,
    r.name AS role_name,
    '' AS project_id,
    '' AS project_slug
FROM roles r
WHERE r.workspace_id = sqlc.arg(workspace_id)
    AND r.name IN (sqlc.slice('role_names'))
UNION ALL
SELECT
    'project' AS resource_type,
    '' AS identity_id,
    '' AS identity_external_id,
    '' AS permission_id,
    '' AS permission_slug,
    '' AS role_id,
    '' AS role_name,
    p.id AS project_id,
    p.slug AS project_slug
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id)
    AND BINARY p.slug = 'default';

-- name: FindLiveKeyForUpdateByID :one
-- Keep this projection small: key updates do not need the RBAC and ratelimit
-- aggregates returned by FindLiveKeyByID.
SELECT
    k.id,
    k.key_auth_id,
    k.hash,
    k.workspace_id,
    k.name,
    k.identity_id,
    a.id AS api_id,
    a.name AS api_name,
    i.external_id AS identity_external_id
FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON k.workspace_id = ws.id
LEFT JOIN identities i ON k.identity_id = i.id AND i.deleted = false
WHERE k.id = sqlc.arg(id)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;

-- name: FindLiveKeyForCreditsByID :one
-- Credit updates only need authorization, audit, cache, and credit state.
SELECT
    k.id,
    k.key_auth_id,
    k.hash,
    k.workspace_id,
    k.name,
    k.refill_day,
    k.refill_amount,
    k.remaining_requests,
    a.id AS api_id
FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON k.workspace_id = ws.id
WHERE k.id = sqlc.arg(id)
    AND k.deleted_at_m IS NULL
    AND a.deleted_at_m IS NULL
    AND ka.deleted_at_m IS NULL
    AND ws.deleted_at_m IS NULL;
