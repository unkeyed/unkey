-- name: FindUnkeyRootKeyByID :one
-- FindUnkeyRootKeyByID reads a new-format root key, excluding soft-deleted keys.
-- It does not fall back to the legacy keys table.
SELECT
    pk,
    id,
    workspace_id,
    hash,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at,
    deleted_at
FROM unkey_root_keys
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: FindUnkeyRootKeyByIDForUpdate :one
-- FindUnkeyRootKeyByIDForUpdate locks a live new-format root key for a mutation.
SELECT
    pk,
    id,
    workspace_id,
    hash,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at,
    deleted_at
FROM unkey_root_keys
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR UPDATE;
