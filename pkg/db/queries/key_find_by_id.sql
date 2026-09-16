-- name: FindKeyByID :one
SELECT
    k.pk, k.id, k.key_auth_id, k.hash, k.prefix, k.start, k.end, k.workspace_id,
    k.for_workspace_id, k.name, k.identity_id, k.meta, k.expires, k.created_at_m,
    k.updated_at_m, k.deleted_at_m, k.refill_day, k.refill_amount,
    k.last_refill_at, k.enabled, k.remaining_requests, k.environment,
    k.last_used_at, k.pending_migration_id
FROM `keys` k
WHERE k.id = sqlc.arg(id);
