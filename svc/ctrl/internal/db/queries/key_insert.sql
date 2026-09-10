-- name: InsertKey :exec
-- InsertKey writes the plaintext key parts and hash in one statement so they stay consistent.
-- Callers that do not know these parts pass empty prefix and end values.
INSERT INTO `keys` (
    id,
    key_auth_id,
    hash,
    prefix,
    start,
    end,
    workspace_id,
    for_workspace_id,
    name,
    identity_id,
    meta,
    expires,
    created_at_m,
    enabled,
    remaining_requests,
    refill_day,
    refill_amount,
    pending_migration_id
) VALUES (
    sqlc.arg(id),
    sqlc.arg(key_space_id),
    sqlc.arg(hash),
    sqlc.arg(prefix),
    sqlc.arg(start),
    sqlc.arg(end),
    sqlc.arg(workspace_id),
    sqlc.arg(for_workspace_id),
    sqlc.arg(name),
    sqlc.arg(identity_id),
    sqlc.arg(meta),
    sqlc.arg(expires),
    sqlc.arg(created_at_m),
    sqlc.arg(enabled),
    sqlc.arg(remaining_requests),
    sqlc.arg(refill_day),
    sqlc.arg(refill_amount),
    sqlc.arg(pending_migration_id)
);
