-- name: InsertUnkeyRootKey :exec
-- InsertUnkeyRootKey creates an administrative credential outside the regular
-- API-key table. Callers insert its permissions and audit events in the same transaction.
INSERT INTO unkey_root_keys (
    id,
    workspace_id,
    key_auth_id,
    for_workspace_id,
    hash,
    name,
    prefix,
    start,
    end,
    enabled,
    expires,
    created_at
) VALUES (
    sqlc.arg(id),
    sqlc.arg(workspace_id),
    sqlc.arg(key_auth_id),
    sqlc.arg(for_workspace_id),
    sqlc.arg(hash),
    sqlc.arg(name),
    sqlc.arg(prefix),
    sqlc.arg(start),
    sqlc.arg(end),
    sqlc.arg(enabled),
    sqlc.arg(expires),
    sqlc.arg(created_at)
);
