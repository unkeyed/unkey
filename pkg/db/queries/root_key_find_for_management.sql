-- name: FindRootKeysForManagement :many
-- FindRootKeysForManagement locates live root keys with one ID in either store.
-- New keys sort first, so updates use them when a migration leaves a legacy twin.
SELECT
    id,
    hash,
    name,
    prefix,
    start,
    enabled,
    expires,
    is_legacy
FROM (
    SELECT
        id,
        hash,
        name,
        prefix,
        start,
        enabled,
        expires,
        FALSE AS is_legacy
    FROM unkey_root_keys
    WHERE unkey_root_keys.id = sqlc.arg(id)
        AND unkey_root_keys.workspace_id = sqlc.narg(workspace_id)
        AND unkey_root_keys.deleted_at IS NULL
    UNION ALL
    SELECT
        id,
        hash,
        name,
        prefix,
        start,
        enabled,
        expires,
        TRUE AS is_legacy
    FROM `keys`
    WHERE `keys`.id = sqlc.arg(id)
        AND `keys`.for_workspace_id = sqlc.narg(workspace_id)
        AND `keys`.deleted_at_m IS NULL
) AS root_keys
ORDER BY is_legacy ASC;
