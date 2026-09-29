-- name: FindRootKeyForAuthentication :one
-- FindRootKeyForAuthentication loads new and legacy root keys in one round trip.
-- Regular API keys are excluded. Legacy assignments and roles are read only for
-- legacy keys; direct permissions are scoped to the target workspace and principal.
WITH root_keys AS (
    SELECT
        id,
        key_auth_id,
        workspace_id,
        for_workspace_id,
        name,
        expires,
        enabled,
        FALSE AS legacy
    FROM unkey_root_keys
    WHERE unkey_root_keys.hash = sqlc.arg(hash) AND unkey_root_keys.deleted_at IS NULL
    UNION ALL
    SELECT
        id,
        key_auth_id,
        workspace_id,
        for_workspace_id,
        name,
        expires,
        enabled,
        TRUE AS legacy
    FROM `keys`
    WHERE `keys`.hash = sqlc.arg(hash) AND `keys`.deleted_at_m IS NULL AND `keys`.for_workspace_id IS NOT NULL
)
SELECT
    k.id,
    k.key_auth_id,
    k.workspace_id,
    k.for_workspace_id,
    k.name,
    k.expires,
    k.enabled,
    a.deleted_at_m AS api_deleted_at_m,
    ws.enabled AS workspace_enabled,
    fws.enabled AS for_workspace_enabled,
    COALESCE(
        (SELECT JSON_ARRAYAGG(slug)
        FROM (
            SELECT slug
            FROM unkey_principal_permissions p
            WHERE p.for_workspace_id = k.for_workspace_id
                AND p.principal_type = 'root_key'
                AND p.principal_id = k.id
            UNION ALL
            SELECT p.slug
            FROM keys_permissions kp
            JOIN permissions p ON p.id = kp.permission_id
            WHERE k.legacy AND kp.key_id = k.id
            UNION ALL
            SELECT p.slug
            FROM keys_roles kr
            JOIN roles_permissions rp ON rp.role_id = kr.role_id
            JOIN permissions p ON p.id = rp.permission_id
            WHERE k.legacy AND kr.key_id = k.id
        ) AS combined_permissions),
        JSON_ARRAY()
    ) AS permissions
FROM root_keys k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id;
