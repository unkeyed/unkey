-- name: ListRootKeyPermissions :many
-- ListRootKeyPermissions loads effective permissions for an authorized page.
-- Legacy role and direct assignments require a legacy root key in the target
-- workspace. Callers deduplicate exact strings, not collation-equivalent strings.
SELECT
    up.principal_id AS key_id,
    up.slug
FROM unkey_principal_permissions up
WHERE up.for_workspace_id = sqlc.narg(for_workspace_id)
    AND up.principal_type = 'root_key'
    AND up.principal_id IN (sqlc.slice(key_ids))
UNION ALL
SELECT
    k.id AS key_id,
    p.slug
FROM `keys` k
JOIN keys_permissions kp ON kp.key_id = k.id
JOIN permissions p ON p.id = kp.permission_id
WHERE k.for_workspace_id = sqlc.narg(for_workspace_id)
    AND k.deleted_at_m IS NULL
    AND k.id IN (sqlc.slice(key_ids))
UNION ALL
SELECT
    k.id AS key_id,
    p.slug
FROM `keys` k
JOIN keys_roles kr ON kr.key_id = k.id
JOIN roles_permissions rp ON rp.role_id = kr.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE k.for_workspace_id = sqlc.narg(for_workspace_id)
    AND k.deleted_at_m IS NULL
    AND k.id IN (sqlc.slice(key_ids));
