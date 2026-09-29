-- name: FindUnkeyRootKeyForAuthentication :one
-- FindUnkeyRootKeyForAuthentication loads an active root key from the new store.
-- Permissions are scoped to the target workspace and root-key principal.
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
        (SELECT JSON_ARRAYAGG(p.slug)
        FROM unkey_principal_permissions p
        WHERE p.for_workspace_id = k.for_workspace_id
            AND p.principal_type = 'root_key'
            AND p.principal_id = k.id),
        JSON_ARRAY()
    ) AS permissions
FROM unkey_root_keys k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
WHERE k.hash = sqlc.arg(hash) AND k.deleted_at IS NULL;

-- name: FindLegacyRootKeyForAuthentication :one
-- FindLegacyRootKeyForAuthentication loads an active root key from the legacy store.
-- It combines principal permissions with legacy direct and role assignments.
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
            SELECT p.slug
            FROM unkey_principal_permissions p
            WHERE p.for_workspace_id = k.for_workspace_id
                AND p.principal_type = 'root_key'
                AND p.principal_id = k.id
            UNION ALL
            SELECT p.slug
            FROM keys_permissions kp
            JOIN permissions p ON p.id = kp.permission_id
            WHERE kp.key_id = k.id
            UNION ALL
            SELECT p.slug
            FROM keys_roles kr
            JOIN roles_permissions rp ON rp.role_id = kr.role_id
            JOIN permissions p ON p.id = rp.permission_id
            WHERE kr.key_id = k.id
        ) AS combined_permissions),
        JSON_ARRAY()
    ) AS permissions
FROM `keys` k
JOIN apis a ON a.key_auth_id = k.key_auth_id
JOIN key_auth ka ON ka.id = k.key_auth_id
JOIN workspaces ws ON ws.id = k.workspace_id
LEFT JOIN workspaces fws ON fws.id = k.for_workspace_id
WHERE k.hash = sqlc.arg(hash)
    AND k.deleted_at_m IS NULL
    AND k.for_workspace_id IS NOT NULL;
