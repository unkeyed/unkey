-- name: ListRootKeys :many
-- ListRootKeys merges both root-key stores into one workspace-scoped ID stream.
-- The cursor is inclusive: a cursor of key_b returns key_b before key_c.
-- Disabled and expired keys remain visible; soft-deleted keys are excluded.
-- A new key hides a legacy key with the same ID during migration.
SELECT
    id,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at
FROM unkey_root_keys
WHERE unkey_root_keys.workspace_id = sqlc.narg(workspace_id)
    AND unkey_root_keys.deleted_at IS NULL
    AND unkey_root_keys.id >= sqlc.arg(id_cursor)
UNION ALL
SELECT
    id,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at_m AS created_at
FROM `keys`
WHERE `keys`.for_workspace_id = sqlc.narg(workspace_id)
    AND `keys`.deleted_at_m IS NULL
    AND `keys`.id >= sqlc.arg(id_cursor)
    AND NOT EXISTS (
        SELECT 1
        FROM unkey_root_keys shadow
        WHERE shadow.id = `keys`.id
            AND shadow.workspace_id = `keys`.for_workspace_id
            AND shadow.deleted_at IS NULL
    )
ORDER BY id ASC
LIMIT ?;
